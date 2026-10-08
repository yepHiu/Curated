"""Prepare and validate a relocatable, source-carrying macOS libmpv bundle.

Homebrew is a build-time input provider only. Installed applications never probe it.
"""
from __future__ import annotations
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tarfile
import tempfile
import time
import urllib.request
from .native_materials import digest, lock_digest

SYSTEM_PREFIXES = ('/System/Library/', '/usr/lib/')
LOCK = 'scripts/release/native-player/macos-arm64-production.json'


def run(*args: str) -> str:
    return subprocess.check_output(args, text=True).strip()


def dependencies(file: Path) -> list[str]:
    return [line.strip().split(' (')[0] for line in run('/usr/bin/otool', '-L', str(file)).splitlines()[1:]]


def system_library(name: str) -> bool:
    return name.startswith(SYSTEM_PREFIXES) and '..' not in Path(name).parts


def safe_file(directory: Path, name: str) -> Path:
    if not name or Path(name).is_absolute() or '..' in Path(name).parts or '\\' in name:
        raise ValueError('Invalid Mac native material path')
    file = (directory / name).resolve()
    if not file.is_relative_to(directory.resolve()):
        raise ValueError('Mac native material escapes bundle')
    return file


def check_file(directory: Path, entry: dict) -> Path:
    file = safe_file(directory, entry['file'])
    if not file.is_file() or digest(file) != entry['sha256']:
        raise ValueError(f"Mac native checksum mismatch: {entry['file']}")
    return file


def fetch(entry: dict, cache: Path) -> Path:
    checksum = entry['sha256']
    if not re.fullmatch('[a-f0-9]{64}', checksum) or not entry['url'].startswith('https://'):
        raise ValueError('Native downloads require pinned SHA256 and HTTPS')
    cache.mkdir(parents=True, exist_ok=True)
    target = cache / checksum
    if target.is_file() and digest(target) == checksum:
        return target
    url = entry['url']
    headers = {'User-Agent': 'Curated-native-builder'}
    if url.startswith('https://ghcr.io/v2/homebrew/core/'):
        repo = url.split('/v2/', 1)[1].split('/blobs/', 1)[0]
        token_url = f'https://ghcr.io/token?service=ghcr.io&scope=repository:{repo}:pull'
        with urllib.request.urlopen(token_url, timeout=60) as response:
            token = json.load(response)['token']
        headers['Authorization'] = 'Bearer ' + token
    # GitHub's archive redirect is equivalent but more reliable in build proxies.
    match = re.fullmatch(r'https://github.com/([^/]+/[^/]+)/archive/(.+)', url)
    if match and match[2].endswith('.tar.gz'):
        url = f'https://codeload.github.com/{match[1]}/tar.gz/{match[2][:-7]}'
    temporary = target.with_suffix('.download')
    for attempt in range(3):
        try:
            with urllib.request.urlopen(urllib.request.Request(url, headers=headers), timeout=120) as response, temporary.open('wb') as output:
                shutil.copyfileobj(response, output)
            if digest(temporary) != checksum:
                raise ValueError('Mac native input SHA256 mismatch: ' + entry['url'])
            temporary.replace(target)
            return target
        except (OSError, ValueError):
            if attempt == 2:
                raise
            time.sleep(2)
        finally:
            temporary.unlink(missing_ok=True)
    raise AssertionError('unreachable')


def validate_bundle(directory: Path, lock: Path | None = None, *, inspect_macho: bool = True) -> dict:
    manifest = json.loads((directory / 'engine-manifest.json').read_text())
    if (manifest.get('schema'), manifest.get('engine'), manifest.get('arch'), manifest.get('platform')) != (1, 'libmpv', 'arm64', 'macos'):
        raise ValueError('Unsupported Mac native bundle')
    if lock and manifest.get('lockSha256') != lock_digest(lock):
        raise ValueError('Mac native bundle differs from release lock')
    runtime = manifest.get('runtime', [])
    names = {item['file'] for item in runtime}
    if not runtime or len(names) != len(runtime) or 'libmpv.dylib' not in names:
        raise ValueError('Mac native runtime incomplete')
    materials = manifest.get('materials', [])
    if not materials or not any(i['file'].startswith('sources/') for i in materials) or not any(i['file'].startswith('licenses/') for i in materials):
        raise ValueError('Mac native source/license materials missing')
    inputs = json.loads(check_file(directory, manifest['buildInputs']).read_text())
    if lock and inputs['lock'] != json.loads(lock.read_text()):
        raise ValueError('Mac native build records differ from release lock')
    expected_names = {i['file'] for i in inputs['lock']['runtime']} | {'libmpv.dylib'}
    if names != expected_names or any(Path(name).name != name for name in names):
        raise ValueError('Mac native library set differs from lock')
    if manifest.get('minimumMacOS') != inputs['lock']['minimumMacOS']:
        raise ValueError('Mac native minimum OS differs from lock')
    for package, info in inputs['lock']['packages'].items():
        recipe = directory / f'metadata/{package}/recipe.rb'
        if not recipe.is_file() or digest(recipe) != info['recipeSha256']:
            raise ValueError('Mac native recipe differs from pinned bottle')
        required = [f'metadata/{package}/recipe.rb', f'metadata/{package}/sbom.spdx.json']
        required += [f"sources/{source['sha256']}" for source in info['sources']]
        listed = {item['file']: item['sha256'] for item in materials}
        if not all(name in listed for name in required):
            raise ValueError('Mac native corresponding materials incomplete')
        if any(listed[f"sources/{s['sha256']}"] != s['sha256'] for s in info['sources']):
            raise ValueError('Mac native source hash differs from lock')
    for item in [*runtime, *materials]:
        file = check_file(directory, item)
        if item in runtime and inspect_macho:
            if run('/usr/bin/lipo', '-archs', str(file)) != 'arm64':
                raise ValueError('Mac native library architecture mismatch')
            for dep in dependencies(file):
                if not system_library(dep) and not (dep.startswith('@loader_path/') and dep.removeprefix('@loader_path/') in names):
                    raise ValueError(f'External Mac native dependency: {dep}')
            # No development-only rpaths may survive relocation.
            lines = run('/usr/bin/otool', '-l', str(file)).splitlines()
            for line in lines:
                if line.strip().startswith('minos '):
                    version = tuple(int(x) for x in (line.split()[1].split('.') + ['0', '0'])[:3])
                    minimum = tuple(int(x) for x in (manifest['minimumMacOS'].split('.') + ['0', '0'])[:3])
                    if version > minimum:
                        raise ValueError('Mac native library requires newer macOS than advertised')
                if line.strip() == 'cmd LC_RPATH':
                    raise ValueError('Development rpath in Mac native runtime')
    return manifest


def prepare(lock: Path, output: Path, cache: Path) -> None:
    if output.exists():
        raise FileExistsError(output)
    frozen = json.loads(lock.read_text())
    output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='curated-mac-engine-', dir=output.parent) as temporary:
        work = Path(temporary)
        bundle = work / 'bundle'
        bundle.mkdir()
        aliases = {}
        for package, info in frozen['packages'].items():
            print(f'Preparing {package} {info["version"]}', flush=True)
            bottle = fetch(info, cache)
            selected = [item for item in frozen['runtime'] if item['package'] == package]
            found = set()
            with tarfile.open(bottle) as archive:
                prefix = f'{package}/{info["version"]}/'
                for member in archive:
                    if member.name.startswith(prefix) and member.issym():
                        alias, target_name = Path(member.name).name, Path(member.linkname).name
                        if alias.endswith('.dylib'):
                            if alias in aliases and aliases[alias] != target_name:
                                raise ValueError('Colliding Mac native library aliases')
                            aliases[alias] = target_name
                    if not member.name.startswith(prefix) or not member.isfile():
                        continue
                    relative = member.name[len(prefix):]
                    target = None
                    for item in selected:
                        if relative == item['path']:
                            target = bundle / item['file']; found.add(item['file'])
                    if relative == f'.brew/{package}.rb':
                        target = bundle / 'metadata' / package / 'recipe.rb'
                    elif relative == 'sbom.spdx.json':
                        target = bundle / 'metadata' / package / 'sbom.spdx.json'
                    elif re.search(r'(^|/)(LICENSE|LICENCE|COPYING|Copyright)([./_-]|$)', relative, re.I):
                        target = safe_file(bundle, f'licenses/{package}/{relative}')
                    if target:
                        target.parent.mkdir(parents=True, exist_ok=True)
                        with archive.extractfile(member) as source, target.open('wb') as destination:
                            shutil.copyfileobj(source, destination)
            if found != {i['file'] for i in selected}:
                raise ValueError(f'Missing libraries in pinned bottle: {package}')
            for source in info['sources']:
                archive = fetch(source, cache)
                target = bundle / 'sources' / source['sha256']
                target.parent.mkdir(exist_ok=True)
                shutil.copyfile(archive, target)
        names = {item['file'] for item in frozen['runtime']}
        # All bundled libraries resolve siblings, never the build machine or user PATH.
        for name in sorted(names):
            file = bundle / name
            os.chmod(file, 0o755)
            architectures = run('/usr/bin/lipo', '-archs', str(file)).split()
            if architectures != ['arm64']:
                thin = file.with_suffix('.thin')
                run('/usr/bin/lipo', str(file), '-thin', 'arm64', '-output', str(thin)); thin.replace(file)
            args = ['/usr/bin/install_name_tool', '-id', '@loader_path/' + name]
            for dep in dependencies(file):
                if system_library(dep):
                    continue
                sibling = Path(dep).name
                visited = set()
                while sibling not in names and sibling in aliases and sibling not in visited:
                    visited.add(sibling)
                    sibling = aliases[sibling]
                if sibling not in names:
                    raise ValueError(f'Unresolved dylib {dep}')
                args += ['-change', dep, '@loader_path/' + sibling]
            lines = run('/usr/bin/otool', '-l', str(file)).splitlines()
            for index, line in enumerate(lines):
                if line.strip() == 'cmd LC_RPATH':
                    args += ['-delete_rpath', lines[index + 2].strip().split(' (')[0].removeprefix('path ')]
            run(*args, str(file))
            run('/usr/bin/codesign', '--force', '--sign', '-', str(file))
        shutil.copy2(bundle / frozen['entry'], bundle / 'libmpv.dylib')
        (bundle / 'licenses/THIRD-PARTY-NOTICES.txt').write_text(
            'Curated bundles libmpv and its dependencies. No external mpv installation is required.\n'
            'Original copyright and license texts are in the component subdirectories.\n'
            'Exact upstream sources and patches are included under sources/ (SHA256 filenames).\n'
            'metadata/ preserves the matching build recipes and SBOMs; build-inputs.json maps\n'
            'every component, input URL, version and checksum. Curated source files retain\n'
            'their own notices; the bundled GPL/LGPL components retain their license terms.\n\n'
            + '\n'.join(f'{name}: {info["version"]} — {info["license"]}' for name, info in frozen['packages'].items()) + '\n')
        shutil.copy2(Path(__file__).parents[1] / 'native-player/MACOS-SOURCES.md', bundle / 'licenses/BUILDING.md')
        (bundle / 'build-inputs.json').write_text(json.dumps({'lock': frozen, 'relocation': '@loader_path siblings; arm64; ad-hoc signing',
            'builder': 'scripts/release/release_lib/macos_native.py'}, indent=2) + '\n')
        def entry(file):
            return {'file': file.relative_to(bundle).as_posix(), 'sha256': digest(file)}
        runtime = [entry(bundle / name) for name in sorted(names | {'libmpv.dylib'})]
        materials = [entry(p) for p in sorted(bundle.rglob('*')) if p.is_file() and p.parent != bundle]
        manifest = {'schema': 1, 'engine': 'libmpv', 'platform': 'macos', 'arch': 'arm64',
            'minimumMacOS': frozen['minimumMacOS'], 'lockSha256': lock_digest(lock), 'runtime': runtime,
            'materials': materials, 'buildInputs': entry(bundle / 'build-inputs.json')}
        (bundle / 'engine-manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
        validate_bundle(bundle, lock)
        bundle.rename(output)


def stage(destination: Path, lock: Path, bundle: Path | None = None) -> dict:
    configured = bundle or Path(os.environ.get('CURATED_NATIVE_MAC_BUNDLE', ''))
    if not configured.is_absolute():
        raise ValueError('Mac Desktop requires an absolute prepared CURATED_NATIVE_MAC_BUNDLE')
    manifest = validate_bundle(configured, lock)
    target = destination / 'native-player-macos'
    target.mkdir(parents=True, exist_ok=False)
    for item in [*manifest['runtime'], *manifest['materials'], manifest['buildInputs']]:
        file = check_file(configured, item)
        output = safe_file(target, item['file'])
        output.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(file, output)
    shutil.copy2(configured / 'engine-manifest.json', target / 'engine-manifest.json')
    return manifest
