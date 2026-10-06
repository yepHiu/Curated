"""Restore the shipped, deduplicated sources and Cargo dependencies without a network."""
import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath
import os
import tempfile
from contextlib import contextmanager
import tarfile
from zipfile import ZipFile


def relative(name):
    path = PurePosixPath(name)
    if path.is_absolute() or '..' in path.parts or '\\' in name or ':' in name:
        raise ValueError('Unsafe source path')
    return path


def restore(archive_path, destination):
    destination.mkdir(parents=True, exist_ok=False)
    with source_reader(archive_path) as archive:
        index = json.loads(archive.read('source-index.json'))
        for identifier, entries in index.items():
            snapshot = destination / 'snapshots' / relative(identifier)
            links = []
            for entry in entries:
                file = snapshot / relative(entry['path'])
                file.parent.mkdir(parents=True, exist_ok=True)
                if 'blob' in entry:
                    data = archive.read('blobs/' + entry['blob'])
                    if hashlib.sha256(data).hexdigest() != entry['blob']:
                        raise ValueError('Source blob checksum mismatch')
                    file.write_bytes(data)
                    file.chmod(entry['mode'])
                else:
                    links.append((file, entry))
            for file, entry in links:
                target = PurePosixPath(entry['link'])
                resolved = (snapshot / target if entry.get('hardlink') else file.parent / target).resolve()
                if '\\' in entry['link'] or ':' in entry['link'] or not resolved.is_relative_to(snapshot.resolve()):
                    raise ValueError('Unsafe source link')
                if entry.get('hardlink'):
                    os.link(resolved, file)
                else:
                    os.symlink(str(target), file)
        vendor = destination / 'vendor'
        vendor.mkdir()
        for crate in json.loads(archive.read('cargo-index.json')):
            import io
            data = archive.read('crates/' + crate['file'])
            if hashlib.sha256(data).hexdigest() != crate['sha256']:
                raise ValueError('Cargo source checksum mismatch')
            with tarfile.open(fileobj=io.BytesIO(data)) as source:
                source.extractall(vendor, filter='data')
            crate_root = vendor / (crate['name'] + '-' + crate['version'])
            hashes = {file.relative_to(crate_root).as_posix(): hashlib.sha256(file.read_bytes()).hexdigest()
                      for file in crate_root.rglob('*') if file.is_file()}
            (crate_root / '.cargo-checksum.json').write_text(json.dumps({'files': hashes, 'package': crate['sha256']}))
        cargo = destination / 'cargo-home'
        cargo.mkdir()
        config = '[source.crates-io]\nreplace-with = "curated-vendor"\n[source.curated-vendor]\ndirectory = ' + json.dumps(str(vendor.resolve())) + '\n[net]\noffline = true\n'
        (cargo / 'config.toml').write_text(config)
    print('Sources restored:', destination.resolve())
    print('For Rust recipes, set CARGO_HOME to', cargo.resolve())


@contextmanager
def source_reader(file):
    if file.is_dir():
        class DirectoryReader:
            def read(self, name):
                return (file / relative(name)).read_bytes()
        yield DirectoryReader()
    elif file.name.endswith('.zip'):
        with ZipFile(file) as archive:
            yield archive
    else:
        try:
            from compression import zstd
            stream = zstd.open(file, 'rb')
        except ImportError:
            import zstandard
            stream = zstandard.ZstdDecompressor().stream_reader(file.open('rb'), closefd=True)
        with tempfile.TemporaryDirectory(prefix='curated-source-blobs-') as temporary:
            with stream, tarfile.open(fileobj=stream, mode='r|') as archive:
                archive.extractall(temporary, filter='data')
            with source_reader(Path(temporary)) as reader:
                yield reader


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('archive', type=Path)
    parser.add_argument('destination', type=Path)
    args = parser.parse_args()
    restore(args.archive, args.destination)
