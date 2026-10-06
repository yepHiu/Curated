"""Optional Windows engine payload. Unprepared development binaries are never staged."""
from __future__ import annotations
from hashlib import file_digest
import json
import os
from pathlib import Path
import shutil
from zipfile import ZipFile
from .pe_imports import imports

WINDOWS_IMPORTS = set(('advapi32 avicap32 avrt bcrypt bcryptprimitives cfgmgr32 crypt32 dnsapi '
    'dwmapi dwrite gdi32 gdiplus imm32 iphlpapi kernel32 msimg32 ncrypt ntdll ole32 oleaut32 '
    'opengl32 rpcrt4 shcore shell32 shlwapi user32 userenv usp10 uxtheme version winmm ws2_32 wsock32').split())


def _entry(root: Path, value: dict) -> Path:
    name = value.get('file')
    digest = value.get('sha256')
    if not isinstance(name, str) or not name or '\\' in name or ':' in name:
        raise ValueError('Invalid native engine manifest path')
    relative = Path(name)
    if relative.is_absolute() or '..' in relative.parts:
        raise ValueError('Native engine path escapes bundle')
    file = (root / relative).resolve()
    if not file.is_relative_to(root.resolve()) or not file.is_file():
        raise ValueError(f'Missing native engine material: {name}')
    with file.open('rb') as stream:
        actual = file_digest(stream, 'sha256').hexdigest()
    if not isinstance(digest, str) or len(digest) != 64 or actual != digest:
        raise ValueError(f'Native engine checksum mismatch: {name}')
    return file


def validate_native_bundle(directory: Path) -> dict:
    manifest = json.loads((directory / 'engine-manifest.json').read_text(encoding='utf-8'))
    if (manifest.get('schema'), manifest.get('engine'), manifest.get('platform'), manifest.get('arch')) != (1, 'mpv', 'windows', 'x64'):
        raise ValueError('Unsupported native engine bundle')
    runtime = manifest.get('runtime', [])
    if not isinstance(runtime, list) or not any(item.get('file') == 'mpv.exe' for item in runtime):
        raise ValueError('Native engine bundle requires mpv.exe')
    for item in runtime:
        _entry(directory, item)
    # 本地的完整对应源码/构建材料随包可离线取得；不把可变化的仓库首页当作源码。
    _entry(directory, manifest.get('correspondingSource', {}))
    _entry(directory, manifest.get('buildInputs', {}))
    licenses = manifest.get('licenses', [])
    if not licenses:
        raise ValueError('Native engine license materials missing')
    for item in licenses:
        _entry(directory, item)
    components = manifest.get('components', [])
    if not isinstance(components, list) or not {'mpv', 'ffmpeg'} <= {item.get('name') for item in components}:
        raise ValueError('Native engine source catalogue incomplete')
    for item in components:
        if not item.get('revision') or not item.get('license') or not item.get('sourcePath'):
            raise ValueError('Native engine source catalogue requires exact revisions and licenses')
    if manifest.get('provider') == 'msys2':
        runtime_names = {item['file'].lower() for item in runtime}
        if len(runtime_names) != len(runtime) or any('/' in name or (name != 'mpv.exe' and not name.endswith('.dll')) for name in runtime_names):
            raise ValueError('Production engine contains duplicate or unexpected runtime files')
        system = set(manifest.get('systemImports', []))
        for name in system:
            if not (name.startswith(('api-ms-', 'ext-ms-')) and name.endswith('.dll')) and name.removesuffix('.dll') not in WINDOWS_IMPORTS:
                raise ValueError('Unknown Windows system import')
        for item in runtime:
            missing = imports(directory / item['file']) - runtime_names - system
            if missing:
                raise ValueError(f"Native engine dependency missing for {item['file']}: {sorted(missing)}")
        with ZipFile(directory / manifest['correspondingSource']['file']) as source:
            index = json.loads(source.read('source-index.json'))
        for component in components:
            if component['sourcePath'].removeprefix('snapshots/') not in index:
                raise ValueError('Native engine component has no corresponding source snapshot')
    return manifest


def stage_native_player(destination: Path, bundle: Path | None = None, *, required: bool = False) -> bool:
    configured = os.environ.get('CURATED_NATIVE_BUNDLE')
    if bundle is None and not configured:
        if required:
            raise ValueError('Windows Desktop requires a prepared CURATED_NATIVE_BUNDLE')
        return False
    source = bundle or Path(configured)
    if not source.is_absolute():
        raise ValueError('CURATED_NATIVE_BUNDLE must be an absolute prepared engine directory')
    manifest = validate_native_bundle(source)
    # 仅复制清单列出的运行时/许可/源码，不能带入未知文件或 Server 的 FFmpeg CLI。
    selected = [*manifest['runtime'], *manifest['licenses'], manifest['correspondingSource'], manifest['buildInputs']]
    target = destination / 'native-player'
    target.mkdir(parents=True, exist_ok=False)
    shutil.copy2(source / 'engine-manifest.json', target / 'engine-manifest.json')
    for item in selected:
        file = _entry(source, item)
        output = target / item['file']
        output.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(file, output)
    validate_native_bundle(target)
    return True
