"""Publish one immutable batch of selected standalone Desktop/Server packages."""
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
from scripts.release.release_lib.latest_release import reconcile_latest
from scripts.release.release_lib.batches import load_batch, selected_components, verify_changes
from scripts.release.release_lib.component_channels import read_channel_file
from scripts.release.release_lib.native_player import source_asset_name, validate_source_reference, validate_source_archive

PATTERN = re.compile(r'(full|server|desktop)-v((?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*))')


def metadata(root: Path, tag: str) -> dict:
    current = versions(root)
    batch = load_batch(root, tag, current)
    if batch:
        commit = legacy.git(root, 'rev-parse', '--verify', f'refs/tags/{tag}^{{commit}}')
        if commit != legacy.git(root, 'rev-parse', 'HEAD'):
            raise ValueError('Checkout is not the immutable tag commit')
        selected = [c for c, m in batch['modules'].items() if m['changed']]
        return {'tag': tag, 'component': 'both' if len(selected) == 2 else selected[0],
                'version': batch['id'], 'commit': commit, 'versions': current, 'batch': batch}
    match = PATTERN.fullmatch(tag)
    if not match:
        raise ValueError('Expected full-v, server-v or desktop-v followed by numeric SemVer')
    component, version = match.groups()
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
    if '## GitHub Release Body' in text:
        text = text.split('## GitHub Release Body', 1)[1].strip()
    if not text:
        raise ValueError('Component release notes must not be empty')
    marker = f"\n<!-- curated-release-batch:{meta['batch']['id']} -->" if 'batch' in meta else ''
    return text + '\n\n' + legacy.source_marker(meta) + marker + '\n'


def require_standalone(component: str) -> None:
    if component not in ('server', 'desktop'):
        raise ValueError('New releases must use server-v or desktop-v; Full is retired')


def module_updates(text: str, meta: dict) -> list[tuple[str, str, str, str]]:
    """Validate the immutable module snapshot shared by notes and display titles."""
    if 'batch' not in meta:
        require_standalone(meta['component'])
    semver = r'(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)'
    rows = re.findall(
        rf'^\|\s*(Desktop|Server)\s*\|\s*(Updated|Unchanged)\s*\|\s*({semver})\s*\|\s*({semver})\s*\|\s*$',
        text, re.MULTILINE)
    if len(rows) != 2 or {row[0] for row in rows} != {'Desktop', 'Server'}:
        raise ValueError('Release notes require Desktop and Server rows: Module | Status | Before | After')
    for name, status, before, after in rows:
        component = name.lower()
        updated = component in selected_components(meta)
        target = meta['batch']['modules'][component]['after'] if 'batch' in meta else meta['version']
        if 'batch' in meta and (before, after) != (meta['batch']['modules'][component]['before'], target):
            raise ValueError('Notes disagree with the prepared batch')
        if updated:
            if status != 'Updated' or after != target or tuple(map(int, after.split('.'))) <= tuple(map(int, before.split('.'))):
                raise ValueError('The released module must be Updated with an increased tag version')
        elif status != 'Unchanged' or before != after:
            raise ValueError('The other module must be Unchanged with the same version')
    return rows


def release_title(text: str, meta: dict) -> str:
    """Use reviewed note snapshots, never unpublished source targets or live channels."""
    if 'batch' in meta:
        module_updates(text, meta)
        return f"Curated {meta['batch']['id']}"
    if meta['component'] == 'full':
        return f"Curated v{meta['version']}"
    after = {name: version for name, _, _, version in module_updates(text, meta)}
    return (f"Curated - Server {after['Server']} + Desktop {after['Desktop']}"
            f" - {meta['component'].title()} update")


def release_body(root: Path, meta: dict) -> str:
    """Require explicit module changes against the currently published channels."""
    text = body(root, meta)
    for name, _, before, _ in module_updates(text, meta):
        component = name.lower()
        updated = component in selected_components(meta)
        published = read_channel(component)
        baseline = published['version'] if published else '0.0.0'
        if updated and before != baseline:
            raise ValueError(f'{name} Before must match the published channel ({baseline})')
        if 'batch' in meta and published and published['sourceCommit'] != meta['batch']['modules'][component]['baselineCommit']:
            raise ValueError(f'{name} source baseline changed; prepare a new batch')
        # Historical component notes allowed older unchanged snapshots. Batches
        # pin both source baselines above to keep scope decisions reproducible.
        if not updated and tuple(map(int, before.split('.'))) > tuple(map(int, baseline.split('.'))):
            raise ValueError(f'Unchanged {name} cannot claim an unpublished version')
    return text


def expected_assets(meta: dict) -> dict[str, dict]:
    components = selected_components(meta)
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
        if 'batch' in meta and entry.get('sourceCommit') != meta['commit']:
            raise ValueError('Batch artifact was built from another commit')
        if 'batch' in meta and 'url' in entry and entry['url'] != f"https://github.com/yepHiu/Curated/releases/download/{meta['tag']}/{name}":
            raise ValueError('Batch artifact URL belongs to another release')


def validate_native_sources(root: Path, meta: dict, entry: dict | None, directories: list[Path]) -> None:
    """新 Windows Desktop 必须与完整源码同批发布；该资产不进入应用更新下载列表。"""
    required = meta.get('batch', {}).get('desktopNativeSources', False)
    if not required:
        if entry is not None:
            raise ValueError('Unexpected native source asset')
        return
    if not entry:
        raise ValueError('Desktop release requires a separate complete native source asset')
    validate_source_reference(entry)
    if (entry['tag'], entry['fileName'], entry.get('sourceCommit')) != (
            meta['tag'], source_asset_name(meta['versions']['desktop']), meta['commit']):
        raise ValueError('Native source asset belongs to another Desktop release')
    paths = [directory / entry['fileName'] for directory in directories if (directory / entry['fileName']).is_file()]
    if len(paths) != 1 or legacy.sha256(paths[0]) != entry['sha256']:
        raise ValueError('Native source asset missing or checksum mismatch')
    manifest = validate_source_archive(paths[0])
    lock = root / 'scripts/release/native-player/windows-x64-production.json'
    if manifest.get('lockSha256') != legacy.sha256(lock):
        raise ValueError('Native source asset does not match the frozen release engine lock')
    from zipfile import ZipFile
    with ZipFile(paths[0]) as archive:
        inputs = json.loads(archive.read(manifest['buildInputs']['file']))
        if inputs.get('lock') != json.loads(lock.read_text(encoding='utf-8')):
            raise ValueError('Native source build records differ from the frozen engine lock')


def stage(root: Path, meta: dict, windows: Path, macos: Path, output: Path) -> None:
    """组装并校验同批所有资产，将独立源码与应用更新清单分开。"""
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
                'format': entry['fileName'].rsplit('.', 1)[1], 'signing': 'ad-hoc', 'notarized': False,
                'sourceCommit': entry.get('sourceCommit', mac['sourceCommit'])})
    validate_entries(meta, entries, [windows, macos])
    native_sources = manifest.get('nativeSources')
    validate_native_sources(root, meta, native_sources, [windows])
    assets = output / 'assets'
    assets.mkdir(parents=True, exist_ok=False)
    for entry in entries:
        name = entry['fileName']
        source = next(d / name for d in (windows, macos) if (d / name).is_file())
        shutil.copy2(source, assets / name)
        entry.setdefault('sourceCommit', meta['commit'])
        entry.setdefault('url', f"https://github.com/yepHiu/Curated/releases/download/{meta['tag']}/{name}")
    if native_sources:
        shutil.copy2(windows / native_sources['fileName'], assets / native_sources['fileName'])
    for component in sorted({e['component'] for e in entries}):
        document = {'schema': 1, 'component': component, 'version': meta['versions'][component],
                    'sourceCommit': meta['commit'], 'artifacts': [e for e in entries if e['component'] == component]}
        (assets / f'{component}.json').write_text(json.dumps(document, indent=2) + '\n')
    (assets / 'release.json').write_text(json.dumps({**meta, 'schema': 1, 'artifacts': entries,
        **({'nativeSources': native_sources} if native_sources else {})}, indent=2) + '\n')
    (assets / 'SHA256SUMS.txt').write_text(''.join(f'{legacy.sha256(p)}  {p.name}\n' for p in sorted(assets.iterdir())))


def verify_distribution(meta: dict, assets: Path, root: Path | None = None) -> dict:
    """上传前验证安装包、通道与独立源码材料，完整成功才允许公开。"""
    manifest = json.loads((assets / 'release.json').read_text())
    if any(manifest.get(k) != v for k, v in meta.items()):
        raise ValueError('Staged source mismatch')
    validate_entries(meta, manifest['artifacts'], [assets])
    native_sources = manifest.get('nativeSources')
    validate_native_sources(root or Path(__file__).resolve().parents[2], meta, native_sources, [assets])
    components = {e['component'] for e in manifest['artifacts']}
    names = set(expected_assets(meta)) | {f'{c}.json' for c in components} | {'release.json', 'SHA256SUMS.txt'}
    if native_sources:
        names.add(native_sources['fileName'])
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
    parent = refs[0]['object']['sha'] if refs else None
    tree_payload = {'tree': [
        {'path': name, 'mode': '100644', 'type': 'blob', 'content': content} for name, content in changes.items()]}
    if parent:
        tree_payload['base_tree'] = legacy.api(f'git/commits/{parent}')['tree']['sha']
    # This branch contains only public manifests, never source or workflows.
    # The initial commit is orphaned, so contents:write does not need workflow scope.
    tree = legacy.api('git/trees', tree_payload)
    next_commit = legacy.api('git/commits', {'message': f"release: advance component channels for {meta['tag']}",
        'tree': tree['sha'], 'parents': [parent] if parent else []})
    if refs:
        legacy.api('git/refs/heads/release-channels', {'sha': next_commit['sha'], 'force': False}, 'PATCH')
    else:
        legacy.api('git/refs', {'ref': 'refs/heads/release-channels', 'sha': next_commit['sha']})


def batch_channels(meta: dict, changes: dict[str, str], assets: Path) -> dict[str, str]:
    """Advance the new feed; keep the legacy Server feed on its reachable bridge."""
    changes = dict(changes)
    if 'server' not in selected_components(meta):
        return changes
    modern = read_channel_file('server', 'server-v2.json')
    document = json.loads((assets / 'server.json').read_text())
    bridge = meta['batch']['serverBridge']
    if bridge and modern is not None and modern != document:
        raise ValueError('Server bridge already exists; never replace the legacy upgrade route')
    if not bridge and modern is None:
        raise ValueError('Publish the Server bridge before date-tagged Server updates')
    content = changes.pop('server.json', None)
    if content is not None:
        changes['server-v2.json'] = content
        if bridge:
            changes['server.json'] = content
    return changes


def check_batch_identity(meta: dict) -> None:
    if 'batch' not in meta:
        return
    marker = f"<!-- curated-release-batch:{meta['batch']['id']} -->"
    page = 1
    while True:
        releases = legacy.api(f'releases?per_page=100&page={page}')
        for release in releases:
            if marker in release.get('body', '') and release['tag_name'] != meta['tag']:
                raise ValueError('Batch date/sequence is already reserved by another release')
        if len(releases) < 100:
            return
        page += 1


def publish(root: Path, meta: dict, output: Path, mode: str) -> None:
    notes = release_body(root, meta)
    assets = output / 'assets'
    verify_distribution(meta, assets, root)
    changes = validate_channel_advance(assets)
    if 'batch' in meta:
        changes = batch_channels(meta, changes, assets)
        check_batch_identity(meta)
    release = legacy.check_release(meta)
    # The tag is already verified by check_release. Omitting target_commitish
    # avoids asking GITHUB_TOKEN to create a tag at a workflow-changing commit.
    payload = {'tag_name': meta['tag'], 'name': release_title(notes, meta),
               'body': notes, 'draft': True, 'prerelease': False, 'make_latest': 'false'}
    release = legacy.api(f"releases/{release['id']}", payload, 'PATCH') if release else legacy.api('releases', payload)
    subprocess.run(['gh', 'release', 'upload', meta['tag'], '--repo', os.environ['GITHUB_REPOSITORY'],
                    '--clobber', *map(str, sorted(assets.iterdir()))], check=True)
    legacy.verify_uploaded(legacy.api(f"releases/{release['id']}"), assets)
    if mode == 'publish':
        legacy.check_release(meta)
        legacy.api(f"releases/{release['id']}", {'draft': False, 'make_latest': 'false'}, 'PATCH')
        advance_channels(changes, meta)
        # Latest is independent of update feeds. Failure can be retried without
        # hiding a public release or replacing already verified assets.
        reconcile_latest(legacy.api)
    print(f"{mode}: {release['html_url']}")


def main() -> None:
    """按不可变标签执行构建、来源校验、发布或已验证资产的恢复。"""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('check', 'windows', 'macos', 'stage', 'publish', 'channels'))
    parser.add_argument('--tag', required=True)
    parser.add_argument('--source-root', type=Path, help='Exact original tag checkout when recovering already verified artifacts')
    parser.add_argument('--mode', choices=('draft', 'publish'), default='draft')
    parser.add_argument('--output', type=Path, default=Path('.workspace/component-cd'))
    parser.add_argument('--windows', type=Path, default=Path('release/windows-components'))
    parser.add_argument('--macos', type=Path, default=Path('release/macos-desktop'))
    args = parser.parse_args()
    # Historical Full manifests can still be recovered, but never built or published anew.
    if args.command != 'channels' and args.tag.startswith('full-v'):
        require_standalone(args.tag.split('-v', 1)[0])
    root = args.source_root.resolve() if args.source_root else Path(__file__).resolve().parents[2]
    if args.source_root and args.command not in ('stage', 'publish', 'channels'):
        raise ValueError('Separate source checkout is only supported for artifact recovery')
    meta = metadata(root, args.tag)
    if args.command != 'channels':
        if 'batch' not in meta:
            raise ValueError('New releases require a prepared batch; run release:prepare')
        verify_changes(root, meta['batch'])
    if args.command == 'check':
        release_body(root, meta)
        check_batch_identity(meta)
        legacy.check_release(meta)
        if target := os.environ.get('GITHUB_OUTPUT'):
            with open(target, 'a') as stream:
                stream.writelines(f'{k}={meta[k]}\n' for k in ('tag', 'component', 'version', 'commit'))
    elif args.command == 'windows':
        package_windows(root, args.windows, meta['component'], release_tag=meta['tag'])
    elif args.command == 'macos':
        if 'desktop' not in selected_components(meta):
            raise ValueError('This batch has no Desktop update')
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
        verify_distribution(meta, args.output / 'assets', root)
        legacy.verify_uploaded(release, args.output / 'assets')
        changes = validate_channel_advance(args.output / 'assets')
        if 'batch' in meta:
            changes = batch_channels(meta, changes, args.output / 'assets')
        advance_channels(changes, meta)
        reconcile_latest(legacy.api)
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
