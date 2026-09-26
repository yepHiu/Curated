"""Build the standalone Apple Silicon Desktop using the locked Electron runtime."""
from __future__ import annotations

import hashlib
import json
import platform
import plistlib
import shutil
import subprocess
import tempfile
from pathlib import Path

from .build_steps import _run, utc_build_stamp
from .components import artifact_name
from .git_utils import resolve_commit
from .versioning import format_version, read_version_state


def desktop_version(root: Path) -> str:
    return format_version(read_version_state(root / 'scripts/release/versions/desktop.json')['current'])


def mac_artifact_names(version: str) -> list[str]:
    return [artifact_name('desktop', version, 'macos', 'arm64', fmt) for fmt in ('dmg', 'zip')]


def stage_app(root: Path, app: Path, version: str, stamp: str) -> None:
    runtime = root / 'node_modules/electron/dist/Electron.app'
    if not runtime.is_dir():
        raise FileNotFoundError('Install the macOS arm64 Electron runtime before packaging')
    if app.exists():
        raise FileExistsError(app)
    # Preserve framework symlinks and executable modes.
    shutil.copytree(runtime, app, symlinks=True)
    contents = app / 'Contents'
    resources = contents / 'Resources'
    payload = resources / 'app'
    payload.mkdir()
    compiled = root / 'electron-dist'
    for required in ('launcher/index.html', 'launcher-preload.cjs', 'main.js', 'preload.cjs', 'connections.html', 'connections-preload.cjs',
                     'connections-ui.js', 'connections.css', 'connections-tokens.css'):
        if not (compiled / required).is_file():
            raise FileNotFoundError(f'Missing Desktop build asset: {required}')
    shutil.copytree(compiled, payload / 'electron-dist', ignore=shutil.ignore_patterns('*.map'))
    metadata = {'schema': 1, 'version': version, 'buildStamp': stamp,
                'distribution': 'desktop', 'updateFeed': 'https://raw.githubusercontent.com/yepHiu/Curated/release-channels/desktop.json'}
    (payload / 'electron-dist/desktop-release.json').write_text(json.dumps(metadata, indent=2) + '\n')
    (payload / 'package.json').write_text(json.dumps({
        'name': 'curated-desktop', 'productName': 'Curated Desktop', 'version': version,
        'type': 'module', 'main': 'electron-dist/main.js',
    }, indent=2) + '\n')
    (payload / 'public').mkdir()
    for name in ('Curated-icon.png', 'Curated-icon-macos.png'):
        shutil.copy2(root / 'public' / name, payload / 'public' / name)
    shutil.copy2(root / 'LICENSE', resources / 'LICENSE-Curated.txt')
    for name in ('LICENSE', 'LICENSES.chromium.html'):
        shutil.copy2(root / 'node_modules/electron/dist' / name, resources / f'Electron-{name}')
    default_app = resources / 'default_app.asar'
    if default_app.exists():
        default_app.unlink()
    executable = contents / 'MacOS/Electron'
    executable.rename(contents / 'MacOS/Curated Desktop')
    info_path = contents / 'Info.plist'
    with info_path.open('rb') as stream:
        info = plistlib.load(stream)
    info.update(CFBundleExecutable='Curated Desktop', CFBundleName='Curated Desktop',
                CFBundleDisplayName='Curated Desktop', CFBundleIdentifier='com.curated.desktop',
                CFBundleShortVersionString=version, CFBundleVersion=version,
                CFBundleIconFile='curated.icns', NSHighResolutionCapable=True)
    with info_path.open('wb') as stream:
        plistlib.dump(info, stream)


def package_macos_desktop(root: Path, output: Path) -> Path:
    if platform.system() != 'Darwin' or platform.machine() != 'arm64':
        raise RuntimeError('macOS Desktop packaging requires an Apple Silicon macOS host')
    version = desktop_version(root)
    names = mac_artifact_names(version)
    output = output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    # Never overwrite existing production packages, even on a same-version rebuild.
    for name in [*names, 'desktop-macos.json']:
        if (output / name).exists():
            raise FileExistsError(f'Release artifact already exists: {output / name}')
    _run(['pnpm', 'build:electron:main'], cwd=root)
    stamp = utc_build_stamp()
    with tempfile.TemporaryDirectory(prefix='curated-desktop-macos-') as temporary:
        work = Path(temporary)
        volume = work / 'volume'
        volume.mkdir()
        app = volume / 'Curated Desktop.app'
        stage_app(root, app, version, stamp)
        binary = app / 'Contents/MacOS/Curated Desktop'
        arch = subprocess.check_output(['lipo', '-archs', str(binary)], text=True).strip()
        if arch != 'arm64':
            raise ValueError(f'Expected arm64 Electron, got {arch}')
        iconset = work / 'curated.iconset'
        iconset.mkdir()
        for size in (16, 32, 128, 256, 512):
            for scale in (1, 2):
                name = f'icon_{size}x{size}' + ('@2x' if scale == 2 else '') + '.png'
                _run(['sips', '-z', str(size * scale), str(size * scale),
                      str(root / 'public/Curated-icon-macos.png'), '--out', str(iconset / name)], cwd=root)
        _run(['iconutil', '-c', 'icns', str(iconset), '-o', str(app / 'Contents/Resources/curated.icns')], cwd=root)
        # Ad-hoc signing supports arm64 execution; this is not Developer ID signing/notarization.
        _run(['codesign', '--force', '--deep', '--sign', '-', str(app)], cwd=root)
        _run(['codesign', '--verify', '--deep', '--strict', str(app)], cwd=root)
        _run(['ditto', '-c', '-k', '--sequesterRsrc', '--keepParent', str(app), str(output / names[1])], cwd=root)
        (volume / 'Applications').symlink_to('/Applications')
        _run(['hdiutil', 'create', '-volname', 'Curated Desktop', '-srcfolder', str(volume),
              '-format', 'UDZO', str(output / names[0])], cwd=root)
        _run(['hdiutil', 'verify', str(output / names[0])], cwd=root)
    artifacts = []
    for name in names:
        with (output / name).open('rb') as stream:
            digest = hashlib.file_digest(stream, 'sha256').hexdigest()
        artifacts.append({'fileName': name, 'sha256': digest})
    manifest = {'schema': 1, 'component': 'desktop', 'version': version, 'buildStamp': stamp,
                'platform': 'macos', 'arch': 'arm64', 'distribution': 'desktop',
                'signing': 'ad-hoc', 'notarized': False,
                'sourceCommit': resolve_commit(root, 'HEAD'), 'artifacts': artifacts}
    (output / 'desktop-macos.json').write_text(json.dumps(manifest, indent=2) + '\n')
    return output
