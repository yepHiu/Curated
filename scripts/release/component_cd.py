"""Build and publish the split distribution without changing the legacy latest feed."""
from __future__ import annotations
import argparse
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
from scripts.release import cd_release as legacy
from scripts.release.release_lib.components import artifact_name
from scripts.release.release_lib.windows_components import versions, package_windows
from scripts.release.release_lib.macos_desktop import package_macos_desktop
from scripts.release.release_lib.component_channels import reuse_assets, read_channel

PATTERN = re.compile(r'(full|server|desktop)-v((?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*))')


def metadata(root: Path, tag: str) -> dict:
    match = PATTERN.fullmatch(tag)
    if not match:
        raise ValueError('Expected full-v, server-v or desktop-v followed by numeric SemVer')
    component, version = match.groups()
    current = versions(root)
    if current[component] != version:
        raise ValueError('Tag disagrees with component version source')
    commit = legacy.git(root, 'rev-parse', '--verify', f'refs/tags/{tag}^{{commit}}')
    if commit != legacy.git(root, 'rev-parse', 'HEAD'):
        raise ValueError('Checkout is not the immutable tag commit')
    return {'tag': tag, 'component': component, 'version': version, 'commit': commit,
            'versions': current}


def body(root: Path, meta: dict) -> str:
    # Component-specific notes avoid collisions between independently versioned products.
    file = root / 'docs/release-notes' / f"{meta['tag']}.md"
    text = file.read_text(encoding='utf-8').strip()
    if not text:
        raise ValueError('Component release notes must not be empty')
    return text + '\n\n' + legacy.source_marker(meta) + '\n'


def legacy_latest() -> dict:
    release = legacy.api('releases/latest')
    if not re.fullmatch(r'v\d+\.\d+\.\d+', release['tag_name']):
        raise ValueError('Legacy latest is not a compatible all-in-one release')
    installers = [a['name'] for a in release['assets'] if a['name'].lower().endswith('.exe')]
    if installers != [f"Curated-Setup-{release['tag_name'][1:]}.exe"]:
        raise ValueError('Legacy latest must expose exactly one compatible installer')
    return release


def expected_assets(meta: dict) -> dict[str, dict]:
    components = ('server', 'desktop', 'full') if meta['component'] == 'full' else (meta['component'],)
    result = {}
    for component in components:
        targets = [('windows', 'x64', 'exe'), ('windows', 'x64', 'zip')]
        if component == 'desktop':
            targets += [('macos', 'arm64', 'dmg'), ('macos', 'arm64', 'zip')]
        for platform, arch, fmt in targets:
            version = meta['versions'][component]
            name = artifact_name(component, version, platform, arch, fmt)
            result[name] = {'component': component, 'variant': 'bundle' if component == 'full' else 'standalone',
                            'version': version, 'channel': 'stable', 'platform': platform, 'arch': arch,
                            'format': fmt, 'fileName': name}
    return result


def validate_entries(meta: dict, entries: list[dict], directories: list[Path]) -> None:
    expected = expected_assets(meta)
    if len(entries) != len(expected) or {e['fileName'] for e in entries} != set(expected):
        raise ValueError('Release is incomplete: all required component/platform packages must exist')
    for entry in entries:
        name = entry['fileName']
        if any(entry.get(k) != v for k, v in expected[name].items()):
            raise ValueError('Component artifact identity mismatch')
        paths = [d / name for d in directories if (d / name).is_file()]
        if len(paths) != 1 or paths[0].stat().st_size == 0 or legacy.sha256(paths[0]) != entry['sha256']:
            raise ValueError('Component artifact missing or checksum mismatch')
        if entry['component'] == 'full' and entry.get('components') != {c: meta['versions'][c] for c in ('server', 'desktop')}:
            raise ValueError('Full must pin the exact component versions')


def stage(root: Path, meta: dict, windows: Path, macos: Path, output: Path) -> None:
    manifest = json.loads((windows / 'windows-components.json').read_text())
    if manifest['sourceCommit'] != meta['commit']:
        raise ValueError('Windows packages were built from another commit')
    entries = manifest['artifacts']
    if meta['component'] != 'server':
        mac = json.loads((macos / 'desktop-macos.json').read_text())
        if any(mac.get(k) != v for k, v in {'sourceCommit': meta['commit'], 'component': 'desktop',
            'version': meta['versions']['desktop'], 'platform': 'macos', 'arch': 'arm64', 'distribution': 'desktop'}.items()):
            raise ValueError('Mac packages do not match the source/version/architecture')
        for entry in mac['artifacts']:
            entries.append({**entry, 'component': 'desktop', 'variant': 'standalone', 'channel': 'stable',
                'version': mac['version'], 'platform': 'macos', 'arch': 'arm64',
                'format': entry['fileName'].rsplit('.', 1)[1], 'signing': 'ad-hoc', 'notarized': False})
    validate_entries(meta, entries, [windows, macos])
    assets = output / 'assets'
    assets.mkdir(parents=True, exist_ok=False)
    for entry in entries:
        name = entry['fileName']
        source = next(d / name for d in (windows, macos) if (d / name).is_file())
        shutil.copy2(source, assets / name)
        entry.setdefault('sourceCommit', meta['commit'])
        entry.setdefault('url', f"https://github.com/yepHiu/Curated/releases/download/{meta['tag']}/{name}")
    for component in sorted({e['component'] for e in entries}):
        document = {'schema': 1, 'component': component, 'version': meta['versions'][component],
                    'sourceCommit': meta['commit'], 'artifacts': [e for e in entries if e['component'] == component]}
        (assets / f'{component}.json').write_text(json.dumps(document, indent=2) + '\n')
    (assets / 'release.json').write_text(json.dumps({**meta, 'schema': 1, 'artifacts': entries}, indent=2) + '\n')
    (assets / 'SHA256SUMS.txt').write_text(''.join(f'{legacy.sha256(p)}  {p.name}\n' for p in sorted(assets.iterdir())))


def verify_distribution(meta: dict, assets: Path) -> dict:
    manifest = json.loads((assets / 'release.json').read_text())
    if any(manifest.get(k) != v for k, v in meta.items()):
        raise ValueError('Staged source mismatch')
    validate_entries(meta, manifest['artifacts'], [assets])
    components = {e['component'] for e in manifest['artifacts']}
    names = set(expected_assets(meta)) | {f'{c}.json' for c in components} | {'release.json', 'SHA256SUMS.txt'}
    if {p.name for p in assets.iterdir()} != names:
        raise ValueError('Unexpected staged assets')
    checksums = ''.join(f'{legacy.sha256(assets / name)}  {name}\n' for name in sorted(names - {'SHA256SUMS.txt'}))
    if (assets / 'SHA256SUMS.txt').read_text() != checksums:
        raise ValueError('Staged checksum list mismatch')
    for component in components:
        document = json.loads((assets / f'{component}.json').read_text())
        expected = {'schema': 1, 'component': component, 'version': meta['versions'][component],
                    'sourceCommit': meta['commit'], 'artifacts': [e for e in manifest['artifacts'] if e['component'] == component]}
        if document != expected:
            raise ValueError('Component feed disagrees with verified distribution')
        for entry in document['artifacts']:
            if not entry['url'].startswith('https://github.com/yepHiu/Curated/releases/download/') or not entry['url'].endswith('/' + entry['fileName']):
                raise ValueError('Component download URL is not an official release asset')
    return manifest


def validate_channel_advance(assets: Path) -> dict[str, str]:
    changes = {}
    for component in ('server', 'desktop', 'full'):
        file = assets / f'{component}.json'
        if not file.exists():
            continue
        current = json.loads(file.read_text())
        previous = read_channel(component)
        if previous:
            old = tuple(map(int, previous['version'].split('.')))
            new = tuple(map(int, current['version'].split('.')))
            if old > new:
                raise ValueError('Refusing to downgrade a published component channel')
            if old == new:
                identity = lambda m: sorted((a['fileName'], a['sha256'], a['url']) for a in m['artifacts'])
                if identity(previous) != identity(current):
                    raise ValueError('Published component versions are immutable; increment the component version')
                continue
        changes[f'{component}.json'] = file.read_text()
    return changes


def advance_channels(changes: dict[str, str], meta: dict) -> None:
    if not changes:
        return
    # Use the Git Data API: a fast-forward-only ref update prevents lost concurrent writes.
    refs = legacy.api('git/matching-refs/heads/release-channels')
    refs = [r for r in refs if r['ref'] == 'refs/heads/release-channels']
    parent = refs[0]['object']['sha'] if refs else meta['commit']
    commit = legacy.api(f'git/commits/{parent}')
    tree = legacy.api('git/trees', {'base_tree': commit['tree']['sha'], 'tree': [
        {'path': name, 'mode': '100644', 'type': 'blob', 'content': content} for name, content in changes.items()]})
    next_commit = legacy.api('git/commits', {'message': f"release: advance component channels for {meta['tag']}",
        'tree': tree['sha'], 'parents': [parent]})
    if refs:
        legacy.api('git/refs/heads/release-channels', {'sha': next_commit['sha'], 'force': False}, 'PATCH')
    else:
        legacy.api('git/refs', {'ref': 'refs/heads/release-channels', 'sha': next_commit['sha']})


def publish(root: Path, meta: dict, output: Path, mode: str) -> None:
    assets = output / 'assets'
    verify_distribution(meta, assets)
    changes = validate_channel_advance(assets)
    old_latest = legacy_latest()
    release = legacy.check_release(meta)
    # The tag is already verified by check_release. Omitting target_commitish
    # avoids asking GITHUB_TOKEN to create a tag at a workflow-changing commit.
    payload = {'tag_name': meta['tag'], 'name': f"Curated {meta['tag']}",
               'body': body(root, meta), 'draft': True, 'prerelease': False, 'make_latest': 'false'}
    release = legacy.api(f"releases/{release['id']}", payload, 'PATCH') if release else legacy.api('releases', payload)
    subprocess.run(['gh', 'release', 'upload', meta['tag'], '--repo', os.environ['GITHUB_REPOSITORY'],
                    '--clobber', *map(str, sorted(assets.iterdir()))], check=True)
    legacy.verify_uploaded(legacy.api(f"releases/{release['id']}"), assets)
    if mode == 'publish':
        legacy.check_release(meta)
        # Explicitly pin the compatible legacy release before adding stable component releases.
        legacy.api(f"releases/{old_latest['id']}", {'make_latest': 'true'}, 'PATCH')
        if legacy_latest()['id'] != old_latest['id']:
            raise ValueError('Legacy latest changed during publication')
        legacy.api(f"releases/{release['id']}", {'draft': False, 'make_latest': 'false'}, 'PATCH')
        if legacy_latest()['id'] != old_latest['id']:
            legacy.api(f"releases/{release['id']}", {'draft': True}, 'PATCH')
            raise ValueError('Legacy feed isolation check failed; component release returned to draft')
        advance_channels(changes, meta)
    print(f"{mode}: {release['html_url']}")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('check', 'windows', 'macos', 'stage', 'publish', 'channels'))
    parser.add_argument('--tag', required=True)
    parser.add_argument('--source-root', type=Path, help='Exact original tag checkout when recovering already verified artifacts')
    parser.add_argument('--mode', choices=('draft', 'publish'), default='draft')
    parser.add_argument('--output', type=Path, default=Path('.workspace/component-cd'))
    parser.add_argument('--windows', type=Path, default=Path('release/windows-components'))
    parser.add_argument('--macos', type=Path, default=Path('release/macos-desktop'))
    args = parser.parse_args()
    root = args.source_root.resolve() if args.source_root else Path(__file__).resolve().parents[2]
    if args.source_root and args.command not in ('stage', 'publish', 'channels'):
        raise ValueError('Separate source checkout is only supported for artifact recovery')
    meta = metadata(root, args.tag)
    if args.command == 'check':
        body(root, meta)
        legacy.check_release(meta)
        legacy_latest()
        if target := os.environ.get('GITHUB_OUTPUT'):
            with open(target, 'a') as stream:
                stream.writelines(f'{k}={meta[k]}\n' for k in ('tag', 'component', 'version', 'commit'))
    elif args.command == 'windows':
        package_windows(root, args.windows, meta['component'])
    elif args.command == 'macos':
        args.macos.mkdir(parents=True, exist_ok=True)
        entries = reuse_assets('desktop', meta['versions']['desktop'], 'macos', 'arm64', args.macos)
        if not entries:
            package_macos_desktop(root, args.macos)
        else:
            (args.macos / 'desktop-macos.json').write_text(json.dumps({'schema': 1, 'component': 'desktop',
                'version': meta['versions']['desktop'], 'platform': 'macos', 'arch': 'arm64',
                'distribution': 'desktop', 'sourceCommit': meta['commit'], 'artifacts': entries}, indent=2) + '\n')
    elif args.command == 'stage':
        stage(root, meta, args.windows, args.macos, args.output)
    elif args.command == 'channels':
        # Explicit recovery after successful publication but a failed channel-ref update.
        release = legacy.find_release(meta['tag'])
        if not release or release['draft'] or legacy.source_marker(meta) not in release['body']:
            raise ValueError('Channel recovery requires a published release from this commit')
        verify_distribution(meta, args.output / 'assets')
        legacy.verify_uploaded(release, args.output / 'assets')
        legacy_latest()
        advance_channels(validate_channel_advance(args.output / 'assets'), meta)
    else:
        publish(root, meta, args.output, args.mode)

if __name__ == '__main__':
    try:
        main()
    except Exception as error:
        import traceback
        detail = traceback.format_exc()[-6000:].replace('%', '%25').replace('\r', '%0D').replace('\n', '%0A')
        print('::error title=Release diagnostics::' + detail, flush=True)
        raise
