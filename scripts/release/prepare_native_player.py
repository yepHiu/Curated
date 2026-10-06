"""Prepare the frozen Windows mpv runtime, sources, build records and notices."""
from __future__ import annotations
import argparse
from concurrent.futures import ThreadPoolExecutor
import io
import json
from pathlib import Path
import sys
import tarfile

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))
from scripts.release.release_lib.native_materials import download, digest, package_archive, stage_package, write_source_archive
from scripts.release.release_lib.native_player import validate_native_bundle


def prepare(lock_path: Path, output: Path, cache: Path) -> Path:
    lock = json.loads(lock_path.read_text(encoding='utf-8'))
    if (lock.get('schema'), lock.get('provider'), lock.get('engine'), lock.get('platform'), lock.get('arch')) != (1, 'msys2', 'mpv', 'windows', 'x64'):
        raise ValueError('Unsupported production engine lock')
    if output.exists():
        manifest = validate_native_bundle(output)
        if manifest.get('lockSha256') != digest(lock_path) or manifest.get('preparationRevision') != 2:
            raise ValueError('Existing bundle uses another engine lock; select a new output directory')
        return output
    inputs = [p['binary'] for p in lock['packages']] + lock['sources'] + lock['crates']
    print(f"Verifying {len(inputs)} frozen engine inputs", flush=True)
    with ThreadPoolExecutor(max_workers=6) as pool:
        for file in pool.map(lambda entry: download(entry, cache), inputs):
            print('Verified ' + file.name, flush=True)
    # Build in a new directory. A partial preparation has no validated final path.
    import tempfile
    output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='native-materials-', dir=output.parent) as temporary:
        bundle = Path(temporary) / 'bundle'
        bundle.mkdir()
        records, runtime, licenses = {}, [], []
        for package in lock['packages']:
            records[package['name']], package_licenses = stage_package(package, cache / package['binary']['file'], bundle)
            runtime.extend({'file': r['file'], 'sha256': r['sha256']} for r in package.get('runtime', []))
            licenses.extend(package_licenses)
        materials = bundle / 'sources'
        materials.mkdir()
        source_archive = materials / 'complete-sources.tar.zst'
        write_source_archive(source_archive, lock['sources'], lock['crates'], cache,
                             ROOT / 'scripts/release/native-player/restore-sources.py')
        build_inputs = materials / 'build-inputs.json'
        build_inputs.write_text(json.dumps({'lock': lock, 'packageRecords': records}, indent=2) + '\n', encoding='utf-8')
        license_root = bundle / 'licenses'
        license_root.mkdir(exist_ok=True)
        # MSYS2 does not install mpv/FFmpeg COPYING; preserve originals from their exact sources.
        for component in ('mpv', 'ffmpeg'):
            source = next(s for s in lock['sources'] if s['id'] == lock['primarySources'][component])
            found = False
            with package_archive(cache / source['file']) as snapshot:
                for member in snapshot:
                    if not member.isfile() or not member.name.endswith(('.tar.gz', '.tar.xz', '.tar.bz2')):
                        continue
                    with tarfile.open(fileobj=io.BytesIO(snapshot.extractfile(member).read())) as project:
                        for original in project:
                            name = Path(original.name).name
                            if original.isfile() and name.startswith(('COPYING', 'LICENSE', 'Copyright')) and len(Path(original.name).parts) <= 2:
                                target = license_root / component / name
                                target.parent.mkdir(exist_ok=True)
                                target.write_bytes(project.extractfile(original).read())
                                licenses.append({'file': target.relative_to(bundle).as_posix(), 'sha256': digest(target)})
                                found = True
            if not found:
                raise ValueError(f'Missing original {component} license')
        notice = license_root / 'THIRD-PARTY-NOTICES.txt'
        notice.write_text((ROOT / 'scripts/release/native-player/THIRD-PARTY-NOTICES.txt').read_text(encoding='utf-8'), encoding='utf-8')
        licenses.append({'file': notice.relative_to(bundle).as_posix(), 'sha256': digest(notice)})
        restore = license_root / 'restore-sources.py'
        restore.write_bytes((ROOT / 'scripts/release/native-player/restore-sources.py').read_bytes())
        licenses.append({'file': restore.relative_to(bundle).as_posix(), 'sha256': digest(restore)})
        components = [{'name': s['name'], 'revision': s['version'], 'license': ', '.join(s['licenses']),
                       'sourcePath': 'snapshots/' + s['id'], 'sourceArchiveSha256': s['sha256']}
                      for s in lock['sources']]
        components.extend({'name': 'cargo:' + c['name'], 'revision': c['version'], 'license': c['license'],
                           'sourcePath': 'vendor/' + c['name'] + '-' + c['version'], 'sourceArchiveSha256': c['sha256']}
                          for c in lock['crates'])
        manifest = {'schema': 1, 'engine': 'mpv', 'platform': 'windows', 'arch': 'x64',
                    'provider': 'msys2', 'version': lock['version'], 'lockSha256': digest(lock_path), 'preparationRevision': 2,
                    'runtime': sorted(runtime, key=lambda r: r['file']), 'licenses': licenses,
                    'systemImports': lock['systemImports'], 'components': components,
                    'correspondingSource': {'file': source_archive.relative_to(bundle).as_posix(), 'sha256': digest(source_archive)},
                    'buildInputs': {'file': build_inputs.relative_to(bundle).as_posix(), 'sha256': digest(build_inputs)}}
        (bundle / 'engine-manifest.json').write_text(json.dumps(manifest, indent=2) + '\n', encoding='utf-8')
        validate_native_bundle(bundle)
        bundle.rename(output)
    return output


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--lock', type=Path, default=ROOT / 'scripts/release/native-player/windows-x64-production.json')
    parser.add_argument('--output', type=Path, default=ROOT / '.workspace/native-player-production')
    parser.add_argument('--cache', type=Path, default=ROOT / '.workspace/native-player-downloads')
    args = parser.parse_args()
    print('Production engine bundle:', prepare(args.lock.resolve(), args.output.resolve(), args.cache.resolve()))
