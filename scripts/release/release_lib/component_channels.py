"""Read immutable component assets from the separate stable channel."""
import hashlib
import json
from pathlib import Path
import urllib.error
import urllib.request

FEED_ROOT = 'https://raw.githubusercontent.com/yepHiu/Curated/release-channels'
ASSET_ROOT = 'https://github.com/yepHiu/Curated/releases/download/'


def read_channel(component: str) -> dict | None:
    try:
        with urllib.request.urlopen(f'{FEED_ROOT}/{component}.json', timeout=30) as response:
            data = response.read(1024 * 1024 + 1)
    except urllib.error.HTTPError as error:
        if error.code == 404:
            return None
        raise
    if len(data) > 1024 * 1024:
        raise ValueError('Component channel too large')
    manifest = json.loads(data)
    if manifest.get('schema') != 1 or manifest.get('component') != component:
        raise ValueError('Invalid component channel')
    return manifest


def reuse_assets(component: str, version: str, platform: str, arch: str, output: Path) -> list[dict]:
    manifest = read_channel(component)
    if not manifest:
        return []
    previous = manifest['version']
    numbers = lambda value: tuple(map(int, value.split('.')))
    if numbers(previous) > numbers(version):
        raise ValueError(f'{component} source version is behind the published channel')
    if previous != version:
        return []
    entries = [a for a in manifest['artifacts'] if a['platform'] == platform and a['arch'] == arch]
    expected = {'exe', 'zip'} if platform == 'windows' else {'dmg', 'zip'}
    if {a['format'] for a in entries} != expected or len(entries) != 2:
        raise ValueError('Published component version is missing a platform; allocate a new version')
    from .components import artifact_name
    for entry in entries:
        name = artifact_name(component, version, platform, arch, entry['format'])
        if entry['fileName'] != name or entry['component'] != component or entry['variant'] != 'standalone' or entry['channel'] != 'stable' or entry['version'] != version or not entry['url'].startswith(ASSET_ROOT) or not entry['url'].endswith('/' + name):
            raise ValueError('Invalid published component asset')
        destination = output / name
        if destination.exists():
            raise FileExistsError(destination)
        with urllib.request.urlopen(entry['url'], timeout=60) as source, destination.open('xb') as target:
            import shutil
            shutil.copyfileobj(source, target)
        with destination.open('rb') as stream:
            if hashlib.file_digest(stream, 'sha256').hexdigest() != entry['sha256']:
                raise ValueError('Reused component checksum mismatch')
    return entries
