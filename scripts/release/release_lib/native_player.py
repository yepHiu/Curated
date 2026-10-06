"""Optional Windows engine payload. Unprepared development binaries are never staged."""
from __future__ import annotations
from hashlib import file_digest
import json
import os
from pathlib import Path
import re
import shutil
from zipfile import ZipFile, ZIP_STORED
from .pe_imports import imports
from .native_materials import read_source_index

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


def source_asset_name(version: str) -> str:
    """按 Desktop 版本命名独立对应源码包，不加入应用更新下载列表。"""
    if not re.fullmatch(r'\d+\.\d+\.\d+', version):
        raise ValueError('Invalid native source version')
    return f'Curated-Desktop-Native-Sources-{version}-windows-x64.zip'


def validate_source_reference(value: dict) -> None:
    """运行包只引用同一正式 Release 中的准确源码资产与校验值。"""
    name = value.get('fileName', '')
    tag = value.get('tag', '')
    if not re.fullmatch(r'Curated-Desktop-Native-Sources-\d+\.\d+\.\d+-windows-x64\.zip', name):
        raise ValueError('Invalid native source asset name')
    if not re.fullmatch(r'(release-\d{8}(?:-(?:[2-9]|[1-9]\d+))?|server-v\d+\.\d+\.\d+)', tag):
        raise ValueError('Invalid native source release tag')
    if value.get('url') != f'https://github.com/yepHiu/Curated/releases/download/{tag}/{name}':
        raise ValueError('Native sources must be available beside the binary release')
    if not re.fullmatch(r'[a-f0-9]{64}', value.get('sha256', '')):
        raise ValueError('Native source asset requires SHA256')


def validate_native_bundle(directory: Path, *, prepared: bool = False) -> dict:
    """校验完整准备资料，或附准确独立源码引用的精简运行目录。"""
    manifest = json.loads((directory / 'engine-manifest.json').read_text(encoding='utf-8'))
    if (manifest.get('schema'), manifest.get('engine'), manifest.get('platform'), manifest.get('arch')) != (1, 'mpv', 'windows', 'x64'):
        raise ValueError('Unsupported native engine bundle')
    runtime = manifest.get('runtime', [])
    if not isinstance(runtime, list) or not any(item.get('file') == 'mpv.exe' for item in runtime):
        raise ValueError('Native engine bundle requires mpv.exe')
    for item in runtime:
        _entry(directory, item)
    separate = manifest.get('distribution') == 'runtime-with-separate-sources'
    if separate:
        if prepared:
            raise ValueError('Prepared native bundle requires local corresponding sources')
        validate_source_reference(manifest.get('sourceAsset', {}))
        # 源码清单仍保留准确 archive/hash，但安装目录无需保存其大归档。
        if not re.fullmatch(r'[a-f0-9]{64}', manifest.get('correspondingSource', {}).get('sha256', '')):
            raise ValueError('Native corresponding source checksum missing')
        source_path = manifest['correspondingSource'].get('file', '')
        if not source_path or '\\' in source_path or ':' in source_path or Path(source_path).is_absolute() or '..' in Path(source_path).parts:
            raise ValueError('Native corresponding source path escapes bundle')
        if (directory / source_path).exists():
            raise ValueError('Runtime package must not include the source archive')
        _entry(directory, manifest.get('sourceNotice', {}))
    else:
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
        if separate:
            return manifest
        index, cargo = read_source_index(directory / manifest['correspondingSource']['file'])
        crates = {c['name'] + '-' + c['version'] for c in cargo}
        for component in components:
            source_path = component['sourcePath']
            if not ((source_path.startswith('snapshots/') and source_path.removeprefix('snapshots/') in index)
                    or (source_path.startswith('vendor/') and source_path.removeprefix('vendor/') in crates)):
                raise ValueError('Native engine component has no corresponding source snapshot')
    return manifest


def stage_native_player(destination: Path, bundle: Path | None = None, *, required: bool = False,
                        source_asset: dict | None = None) -> bool:
    """只复制运行依赖和许可；正式包通过独立源码资产引用避免携带大归档。"""
    configured = os.environ.get('CURATED_NATIVE_BUNDLE')
    if bundle is None and not configured:
        if required:
            raise ValueError('Windows Desktop requires a prepared CURATED_NATIVE_BUNDLE')
        return False
    source = bundle or Path(configured)
    if not source.is_absolute():
        raise ValueError('CURATED_NATIVE_BUNDLE must be an absolute prepared engine directory')
    manifest = validate_native_bundle(source, prepared=True)
    # 仅复制清单列出的运行时/许可/源码，不能带入未知文件或 Server 的 FFmpeg CLI。
    selected = [*manifest['runtime'], *manifest['licenses'], manifest['buildInputs']]
    if source_asset is None:
        selected.append(manifest['correspondingSource'])
    else:
        validate_source_reference(source_asset)
    target = destination / 'native-player'
    target.mkdir(parents=True, exist_ok=False)
    for item in selected:
        file = _entry(source, item)
        output = target / item['file']
        output.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(file, output)
    if source_asset is not None:
        from .native_materials import digest
        third_party = target / 'licenses/THIRD-PARTY-NOTICES.txt'
        if third_party.is_file():
            # 准备目录可复用旧 pin 的完整离线资料；运行副本的说明必须与精简交付一致。
            text = third_party.read_text(encoding='utf-8')
            text = text.replace('Complete corresponding sources are shipped locally in\nsources/complete-sources.tar.zst.',
                                'Complete corresponding sources are provided in the separate Native Sources ZIP\n'
                                'alongside the Desktop download. See SOURCE-DOWNLOAD.txt for its exact URL and SHA256.\n'
                                'The ZIP contains sources/complete-sources.tar.zst.')
            text = text.replace('No internet access is needed to obtain these sources.',
                                'After downloading the source ZIP, restoration and building can be performed offline.')
            third_party.write_text(text, encoding='utf-8')
            manifest = {**manifest, 'licenses': [
                {**item, 'sha256': digest(third_party)} if item['file'] == 'licenses/THIRD-PARTY-NOTICES.txt' else item
                for item in manifest['licenses']]}
        notice = target / 'licenses/SOURCE-DOWNLOAD.txt'
        notice.write_text('Complete corresponding sources for this bundled engine are available, at no charge,\n'
                          'alongside this Desktop binary in the same GitHub Release. No source download is\n'
                          'needed for playback. Download and extract the following ZIP for the exact source\n'
                          'archive, build records, licenses and restoration instructions.\n\n'
                          + source_asset['url'] + '\nSHA256: ' + source_asset['sha256'] + '\n', encoding='utf-8')
        manifest = {**manifest, 'distribution': 'runtime-with-separate-sources', 'sourceAsset': source_asset,
                    'sourceNotice': {'file': notice.relative_to(target).as_posix(), 'sha256': digest(notice)}}
    (target / 'engine-manifest.json').write_text(json.dumps(manifest, indent=2) + '\n', encoding='utf-8')
    validate_native_bundle(target)
    return True


def stage_native_sources(bundle: Path, output: Path, version: str, tag: str) -> dict:
    """完整源码作为同一 Release 的独立 ZIP 提供；已有压缩源码采用存储避免重复压缩。"""
    from .native_materials import digest
    manifest = validate_native_bundle(bundle, prepared=True)
    name = source_asset_name(version)
    destination = output / name
    if destination.exists():
        raise FileExistsError(destination)
    output.mkdir(parents=True, exist_ok=True)
    with ZipFile(destination, 'w', ZIP_STORED) as archive:
        archive.write(bundle / 'engine-manifest.json', 'engine-manifest.json')
        for item in [manifest['correspondingSource'], manifest['buildInputs'], *manifest['licenses']]:
            archive.write(_entry(bundle, item), item['file'])
    result = {'fileName': name, 'tag': tag, 'sha256': digest(destination),
              'url': f'https://github.com/yepHiu/Curated/releases/download/{tag}/{name}'}
    validate_source_reference(result)
    validate_source_archive(destination)
    return result


def validate_source_archive(file: Path) -> dict:
    """核对独立包的完整材料及成员 SHA；禁止未知文件、重复成员或缺失源码。"""
    import hashlib
    with ZipFile(file) as archive:
        manifest = json.loads(archive.read('engine-manifest.json'))
        if manifest.get('distribution') or manifest.get('engine') != 'mpv':
            raise ValueError('Source asset requires the complete prepared engine manifest')
        entries = [manifest['correspondingSource'], manifest['buildInputs'], *manifest['licenses']]
        for entry in entries:
            name = entry.get('file', '')
            if not name or '\\' in name or ':' in name or Path(name).is_absolute() or '..' in Path(name).parts:
                raise ValueError('Native source archive path escapes bundle')
        names = ['engine-manifest.json', *[item['file'] for item in entries]]
        if sorted(archive.namelist()) != sorted(names) or len(set(names)) != len(names):
            raise ValueError('Native source asset contains missing or unexpected materials')
        for entry in entries:
            with archive.open(entry['file']) as stream:
                if hashlib.file_digest(stream, 'sha256').hexdigest() != entry['sha256']:
                    raise ValueError('Native source asset material checksum mismatch')
        inputs = json.loads(archive.read(manifest['buildInputs']['file']))
        if manifest.get('provider') == 'msys2':
            lock = inputs['lock']
            if lock.get('engine') != 'mpv' or lock.get('version') != manifest.get('version'):
                raise ValueError('Native source build inputs differ from the engine')
    return manifest
