"""Independent Windows payloads and an offline Full bootstrapper.

Only temporary build trees are cleaned; existing distribution files are immutable.
"""
from __future__ import annotations
import json
import os
from pathlib import Path
import platform
import shutil
import tempfile
from zipfile import ZipFile, ZIP_DEFLATED

from .build_steps import (build_frontend, build_backend, build_electron_main,
                         _bundle_ffmpeg_runtime, _find_iscc, _run, utc_build_stamp)
from .components import artifact_name, component_plan
from .git_utils import resolve_commit

FEED_ROOT = 'https://raw.githubusercontent.com/yepHiu/Curated/release-channels'
IDENTITIES = {'server': 'Curated.Server', 'desktop': 'Curated.Desktop'}


def versions(root: Path) -> dict[str, str]:
    return {c: component_plan(root, c, 'windows', 'x64', 'exe')['version']
            for c in ('server', 'desktop', 'full')}


def validate_payload(directory: Path, component: str) -> None:
    files = {p.relative_to(directory).as_posix().lower() for p in directory.rglob('*') if p.is_file()}
    if component == 'desktop':
        required = {'curated desktop.exe', 'resources/app/electron-dist/main.js', 'resources/app/package.json'}
        if any('frontend-dist/' in p or 'third_party/' in p or p.endswith(('/curated.exe', '/ffmpeg.exe', '/ffprobe.exe')) for p in files):
            raise ValueError('Desktop payload contains Server dependencies')
        metadata = json.loads((directory / 'resources/app/electron-dist/desktop-release.json').read_text())
        if metadata['distribution'] != 'desktop':
            raise ValueError('Desktop metadata must be standalone')
    elif component == 'server':
        required = {'curated.exe', 'frontend-dist/index.html', 'third_party/ffmpeg/bin/ffmpeg.exe', 'third_party/ffmpeg/bin/ffprobe.exe'}
        if any('electron' in p or p.endswith('.asar') or p.startswith('resources/app/') for p in files):
            raise ValueError('Server payload contains Electron')
    else:
        raise ValueError('Only independent components have runtime payloads')
    if not required <= files:
        raise ValueError(f'Incomplete {component} payload: {required - files}')


def stage_desktop(root: Path, destination: Path, version: str, stamp: str) -> None:
    shutil.copytree(root / 'node_modules/electron/dist', destination)
    (destination / 'electron.exe').rename(destination / 'Curated Desktop.exe')
    (destination / 'resources/default_app.asar').unlink(missing_ok=True)
    payload = destination / 'resources/app'
    shutil.copytree(root / 'electron-dist', payload / 'electron-dist')
    (payload / 'package.json').write_text(json.dumps({'name': 'curated-desktop', 'productName': 'Curated Desktop',
        'version': version, 'main': 'electron-dist/main.js', 'type': 'module'}) + '\n')
    metadata = payload / 'electron-dist/desktop-release.json'
    metadata.write_text(json.dumps({'schema': 1, 'version': version, 'buildStamp': stamp,
        'distribution': 'desktop', 'updateFeed': f'{FEED_ROOT}/desktop.json'}) + '\n')
    (payload / 'public').mkdir()
    shutil.copy2(root / 'public/Curated-icon.png', payload / 'public/Curated-icon.png')
    shutil.copy2(root / 'backend/internal/assets/curated.ico', destination / 'curated.ico')
    shutil.copy2(root / 'LICENSE', destination / 'LICENSE-Curated.txt')
    validate_payload(destination, 'desktop')


def compile_installer(root: Path, work: Path, output: Path, component: str, version: str,
                      values: dict[str, str]) -> Path:
    compiler = _find_iscc()
    if compiler is None:
        raise FileNotFoundError('Inno Setup is required; refusing a partial release')
    name = artifact_name(component, version, 'windows', 'x64', 'exe')
    template = 'Full.iss.tpl' if component == 'full' else 'Component.iss.tpl'
    script = (root / 'scripts/release/windows' / template).read_text()
    values = {'VERSION': version, 'COMPONENT': component.title(), 'OUTPUT': str(output),
              'BASENAME': name[:-4], **values}
    for key, value in values.items():
        script = script.replace(f'__{key}__', value)
    import re
    if re.search(r'__[A-Z_]+__', script):
        raise ValueError('Unresolved installer template field')
    source = work / f'{component}.iss'
    source.write_text(script, encoding='utf-8-sig')
    _run([str(compiler), str(source)], cwd=root)
    if not (output / name).is_file():
        raise FileNotFoundError(name)
    return output / name


def package_windows(root: Path, output: Path, component: str = 'full') -> Path:
    if platform.system() != 'Windows' or platform.machine().lower() not in ('amd64', 'x86_64'):
        raise RuntimeError('Windows components require a Windows x64 host')
    selected = ['server', 'desktop'] if component == 'full' else [component]
    if component not in ('server', 'desktop', 'full'):
        raise ValueError('Unknown component')
    current = versions(root)
    output = output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    names = [artifact_name(c, current[c], 'windows', 'x64', f)
             for c in selected + (['full'] if component == 'full' else []) for f in ('exe', 'zip')]
    for name in names + ['windows-components.json']:
        if (output / name).exists():
            raise FileExistsError(output / name)
    stamp = utc_build_stamp()
    reused = {}
    with tempfile.TemporaryDirectory(prefix='curated-components-') as temporary:
        work = Path(temporary)
        for c in selected:
            from .component_channels import reuse_assets
            entries = reuse_assets(c, current[c], 'windows', 'x64', output)
            if entries:
                reused[c] = entries
                continue
            payload = work / c
            if c == 'server':
                frontend = build_frontend(current[c], str(work / 'frontend'))
                binary = build_backend(current[c], stamp, str(work / 'backend'), distribution='server')
                payload.mkdir()
                shutil.copy2(binary, payload / 'curated.exe')
                shutil.copytree(frontend, payload / 'frontend-dist')
                _bundle_ffmpeg_runtime(root, payload)
                shutil.copy2(root / 'backend/internal/assets/curated.ico', payload / 'curated.ico')
                shutil.copy2(root / 'LICENSE', payload / 'LICENSE-Curated.txt')
                (payload / 'Start Server.cmd').write_text('@echo off\r\nstart "Curated Server" "%~dp0curated.exe" -mode tray\r\n')
                validate_payload(payload, c)
            else:
                build_electron_main()
                stage_desktop(root, payload, current[c], stamp)
            compile_installer(root, work, output, c, current[c], {
                'APP_ID': IDENTITIES[c], 'SOURCE': str(payload),
                'EXE': 'curated.exe' if c == 'server' else 'Curated Desktop.exe',
                'PARAMS': '-mode tray -autostart' if c == 'server' else '',
                'MANAGED_DELETE': 'frontend-dist' if c == 'server' else 'resources\\app',
                'LEGACY_CHECK': 'True' if c == 'server' else 'False',
                'RUN_FLAGS': 'nowait' if c == 'server' else 'nowait postinstall skipifsilent',
            })
            with ZipFile(output / artifact_name(c, current[c], 'windows', 'x64', 'zip'), 'w', ZIP_DEFLATED) as archive:
                for file in sorted(payload.rglob('*')):
                    if file.is_file():
                        archive.write(file, file.relative_to(payload))
        if component == 'full':
            helper = work / 'curated-migrate.exe'
            _run(['go', 'build', '-tags', 'release', '-o', str(helper), './cmd/curated-migrate'], cwd=root / 'backend')
            installers = {c: output / artifact_name(c, current[c], 'windows', 'x64', 'exe') for c in selected}
            compile_installer(root, work, output, 'full', current['full'], {
                'SERVER_INSTALLER': str(installers['server']), 'DESKTOP_INSTALLER': str(installers['desktop']),
                'MIGRATION_HELPER': str(helper),
                'SERVER_VERSION': current['server'], 'DESKTOP_VERSION': current['desktop'],
            })
            # Full ZIP is the offline installer kit, not a second coupled runtime.
            with ZipFile(output / artifact_name('full', current['full'], 'windows', 'x64', 'zip'), 'w', ZIP_DEFLATED) as archive:
                for installer in installers.values():
                    archive.write(installer, installer.name)
                bootstrap = output / artifact_name('full', current['full'], 'windows', 'x64', 'exe')
                archive.write(bootstrap, bootstrap.name)
                archive.writestr('README.txt', 'Run Curated-Full-Setup to install both independent components.\nServer owns your data. Exiting Desktop does not stop Server.\n')
    from hashlib import file_digest
    artifacts = []
    for c in selected + (['full'] if component == 'full' else []):
        for fmt in ('exe', 'zip'):
            name = artifact_name(c, current[c], 'windows', 'x64', fmt)
            with (output / name).open('rb') as stream:
                digest = file_digest(stream, 'sha256').hexdigest()
            previous = next((e for e in reused.get(c, []) if e['format'] == fmt), {})
            entry = {**previous, 'component': c, 'variant': 'bundle' if c == 'full' else 'standalone',
                     'version': current[c], 'channel': 'stable', 'platform': 'windows', 'arch': 'x64',
                     'format': fmt, 'fileName': name, 'sha256': digest,
                     'sourceCommit': previous.get('sourceCommit', resolve_commit(root, 'HEAD'))}
            if c == 'full':
                entry['components'] = {k: current[k] for k in selected}
            artifacts.append(entry)
    (output / 'windows-components.json').write_text(json.dumps({'schema': 1,
        'sourceCommit': resolve_commit(root, 'HEAD'), 'buildStamp': stamp, 'artifacts': artifacts}, indent=2) + '\n')
    return output
