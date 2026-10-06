"""Prepare only frozen, source-complete third-party engine inputs."""
from __future__ import annotations
from contextlib import contextmanager
import io
import hashlib
import json
import posixpath
from pathlib import Path, PurePosixPath
import shutil
import subprocess
import tarfile
import tempfile
import time
import urllib.request
from zipfile import ZipFile, ZipInfo, ZIP_STORED, ZIP_DEFLATED


def digest(file: Path) -> str:
    with file.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def download(entry: dict, cache: Path) -> Path:
    name = entry['file']
    if PurePosixPath(name).name != name or '\\' in name or ':' in name:
        raise ValueError('Invalid download cache name')
    file = cache / name
    if file.is_file() and digest(file) == entry['sha256']:
        return file
    if not entry['url'].startswith('https://'):
        raise ValueError('Engine inputs require HTTPS')
    cache.mkdir(parents=True, exist_ok=True)
    temporary = file.with_suffix(file.suffix + '.download')
    for attempt in range(4):
        try:
            with urllib.request.urlopen(entry['url'], timeout=120) as response, temporary.open('wb') as output:
                shutil.copyfileobj(response, output, length=1024 * 1024)
            if digest(temporary) != entry['sha256']:
                raise ValueError(f'Engine input checksum mismatch: {name}')
            temporary.replace(file)
            return file
        except (OSError, ValueError):
            temporary.unlink(missing_ok=True)
            if attempt == 3:
                raise
            time.sleep(2)
    raise AssertionError('unreachable')


@contextmanager
def package_archive(file: Path):
    if not file.name.endswith('.zst'):
        with tarfile.open(file, mode='r|*') as archive:
            yield archive
        return
    # Python 3.14 includes Zstandard; CD uses a pinned wheel on Python 3.12.
    try:
        from compression import zstd
        stream = zstd.open(file, 'rb')
    except ImportError:
        import zstandard
        stream = zstandard.ZstdDecompressor().stream_reader(file.open('rb'), closefd=True)
    with stream, tarfile.open(fileobj=stream, mode='r|') as archive:
        yield archive


def safe_relative(name: str) -> PurePosixPath:
    path = PurePosixPath(name)
    if path.is_absolute() or '..' in path.parts or '\\' in name or ':' in name:
        raise ValueError('Source archive path escapes snapshot')
    return path


@contextmanager
def source_writer(output: Path):
    if output.name.endswith('.zip'):
        with ZipFile(output, 'x', compression=ZIP_STORED, allowZip64=True) as archive:
            def write(name, data):
                entry = ZipInfo(name, date_time=(1980, 1, 1, 0, 0, 0))
                entry.compress_type = ZIP_DEFLATED
                archive.writestr(entry, data, compresslevel=1)
            yield write
        return
    with output.open('xb') as raw:
        try:
            from compression import zstd
            compressed = zstd.open(raw, 'wb', level=16)
        except ImportError:
            import zstandard
            compressed = zstandard.ZstdCompressor(level=16).stream_writer(raw, closefd=False)
        with compressed, tarfile.open(fileobj=compressed, mode='w|') as archive:
            def write(name, data):
                if isinstance(data, str):
                    data = data.encode('utf-8')
                entry = tarfile.TarInfo(name)
                entry.size, entry.mode = len(data), 0o644
                archive.addfile(entry, io.BytesIO(data))
            yield write


def read_source_index(file: Path) -> tuple[dict, list]:
    if file.name.endswith('.zip'):
        with ZipFile(file) as source:
            return json.loads(source.read('source-index.json')), json.loads(source.read('cargo-index.json'))
    values = {}
    with package_archive(file) as source:
        for member in source:
            if member.name in ('source-index.json', 'cargo-index.json'):
                values[member.name] = json.loads(source.extractfile(member).read())
                if len(values) == 2:
                    return values['source-index.json'], values['cargo-index.json']
    raise ValueError('Source archive catalogue missing')


@contextmanager
def source_members(source: dict, cache: Path):
    snapshot = source.get('gitSnapshot')
    if not snapshot:
        with package_archive(cache / source['file']) as archive:
            yield ((member, archive.extractfile(member) if member.isfile() else None) for member in archive)
        return
    # A pinned shallow repository retains the exact commit/tree and the history
    # needed for pkgver(), without redistributing unrelated decades of history.
    with tempfile.TemporaryDirectory(prefix='curated-source-git-') as temporary:
        directory = Path(temporary)
        original = directory / 'original'
        original.mkdir()
        with package_archive(cache / source['file']) as archive:
            archive.extractall(original, filter='data')
        repository = original / safe_relative(snapshot['path'])
        revision = snapshot['revision']
        git = lambda location, *args: subprocess.check_output(['git', '-C', str(location), *args], text=True).strip()
        if git(repository, 'rev-parse', revision + '^{commit}') != revision:
            raise ValueError('Pinned Git source revision missing')
        git(repository, 'update-ref', 'refs/heads/curated-source', revision)
        reduced = directory / 'reduced'
        subprocess.run(['git', 'clone', '--quiet', '--bare', '--single-branch', '--branch', 'curated-source',
                        '--depth=' + str(snapshot['depth']), repository.resolve().as_uri(), str(reduced)], check=True)
        # Do not preserve local temporary paths in the redistributed repository.
        shutil.copy2(repository / 'config', reduced / 'config')
        git(reduced, 'config', 'core.abbrev', str(snapshot['abbrev']))
        if git(reduced, 'describe', '--long', '--tags', revision) != snapshot['describe']:
            raise ValueError('Reduced Git source changes the upstream build version')
        original_tree = git(repository, 'rev-parse', revision + '^{tree}')
        if git(reduced, 'rev-parse', revision + '^{tree}') != original_tree:
            raise ValueError('Reduced Git source changes the corresponding source tree')
        result = directory / 'snapshot.tar'
        with tarfile.open(result, 'w') as target:
            for file in sorted(original.rglob('*')):
                if file.is_file() and not file.is_relative_to(repository):
                    target.add(file, arcname=file.relative_to(original).as_posix(), recursive=False)
            for file in sorted(reduced.rglob('*')):
                if file.is_file():
                    name = snapshot['path'] + '/' + file.relative_to(reduced).as_posix()
                    target.add(file, arcname=name, recursive=False)
            data = json.dumps({'originalArchiveSha256': source['sha256'], **snapshot}, indent=2).encode()
            entry = tarfile.TarInfo(source['base'] + '/CURATED-SOURCE-SNAPSHOT.json')
            entry.size = len(data)
            target.addfile(entry, io.BytesIO(data))
        with tarfile.open(result, 'r|') as archive:
            yield ((member, archive.extractfile(member) if member.isfile() else None) for member in archive)


def write_source_archive(output: Path, sources: list[dict], crates: list[dict], cache: Path,
                         restore_script: Path) -> None:
    """Preserve exact build snapshots and patches; deduplicate identical inputs."""
    index, blobs = {}, set()
    with source_writer(output) as write:
        for source in sources:
            safe_relative(source['id'])
            entries = []
            with source_members(source, cache) as original:
                for member, stream in original:
                    safe_relative(member.name)
                    if member.isfile():
                        data = stream.read()
                        key = hashlib.sha256(data).hexdigest()
                        if key not in blobs:
                            # Upstream tarballs/Git packfiles are already compressed.
                            write('blobs/' + key, data)
                            blobs.add(key)
                        entries.append({'path': member.name, 'blob': key, 'mode': member.mode})
                    elif member.issym() or member.islnk():
                        target = member.linkname if member.islnk() else posixpath.join(posixpath.dirname(member.name), member.linkname)
                        safe_relative(posixpath.normpath(target))
                        entries.append({'path': member.name, 'link': member.linkname,
                                        'hardlink': member.islnk()})
                    elif not member.isdir():
                        raise ValueError('Unsupported source archive member')
            index[source['id']] = entries
        for crate in crates:
            safe_relative(crate['file'])
            write('crates/' + crate['file'], (cache / crate['file']).read_bytes())
        write('source-index.json', json.dumps(index, indent=2) + '\n')
        write('cargo-index.json', json.dumps(crates, indent=2) + '\n')
        write('restore-sources.py', restore_script.read_bytes())


def stage_package(package: dict, file: Path, destination: Path) -> tuple[dict, list[dict]]:
    runtime = {entry['archivePath']: entry for entry in package.get('runtime', [])}
    records, licenses, found = {}, [], set()
    with package_archive(file) as archive:
        for member in archive:
            if not member.isfile():
                continue
            name = member.name
            if name in ('.BUILDINFO', '.PKGINFO'):
                records[name] = archive.extractfile(member).read().decode('utf-8')
            elif name in runtime or name.startswith('ucrt64/share/licenses/'):
                data = archive.extractfile(member).read()
                relative = runtime[name]['file'] if name in runtime else 'licenses/packages/' + name.removeprefix('ucrt64/share/licenses/')
                safe_relative(relative)
                target = destination / relative
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_bytes(data)
                entry = {'file': relative, 'sha256': hashlib.sha256(data).hexdigest()}
                if name in runtime:
                    if entry['sha256'] != runtime[name]['sha256']:
                        raise ValueError('Runtime file differs from frozen package')
                    found.add(name)
                else:
                    licenses.append(entry)
    if found != set(runtime) or set(records) != {'.BUILDINFO', '.PKGINFO'}:
        raise ValueError('Incomplete engine package')
    expected = 'pkgbuild_sha256sum = ' + package['recipeSha256']
    if expected not in records['.BUILDINFO'].splitlines():
        raise ValueError('Binary build recipe differs from corresponding source')
    return records, licenses
