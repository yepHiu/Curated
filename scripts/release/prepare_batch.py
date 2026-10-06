"""Inspect component changes; --write prepares local versions, batch and release notes.

Never creates tags, commits, pushes, builds or publishes. Review generated notes first.
"""
import argparse
import json
from pathlib import Path
import sys
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from scripts.release.release_lib import batches
from scripts.release.release_lib.component_channels import read_channel, read_channel_file
from scripts.release.release_lib.components import artifact_name
from scripts.release.release_lib.native_player import source_asset_name

ROOT = Path(__file__).resolve().parents[2]


def plan(root: Path, date: str | None = None, bumps: dict | None = None) -> dict | None:
    """根据公共基线规划不可变批次，新 Windows Desktop 明确要求独立源码资产。"""
    # Account for remote tags without changing local refs or guessing from stale fetches.
    remote = batches.git(root, 'ls-remote', '--tags', 'origin')
    tags = {line.split('refs/tags/', 1)[1].removesuffix('^{}') for line in remote.splitlines() if 'refs/tags/' in line}
    tags.update(batches.git(root, 'tag', '--list').splitlines())
    existing = []
    for tag in tags:
        if tag.startswith('release-'):
            existing.append(tag.removeprefix('release-'))
    for file in (root / 'scripts/release/batches').glob('*.json'):
        existing.append(json.loads(file.read_text())['id'])
    identifier = batches.next_id(existing, date)
    modules = {}
    for component in batches.MODULES:
        baseline = read_channel(component)
        if not baseline:
            raise ValueError(f'Missing published {component} baseline; initialize explicitly')
        paths = sorted(p for p in batches.changed_paths(root, baseline['sourceCommit']) if component in batches.affected(p))
        before = baseline['version']
        modules[component] = {'before': before, 'after': batches.bump(before, (bumps or {}).get(component, 'patch')) if paths else before,
                              'changed': bool(paths), 'baselineCommit': baseline['sourceCommit'], 'paths': paths,
                              'inputDigest': batches.input_digest(root, baseline['sourceCommit'], paths),
                              'previousArtifacts': baseline['artifacts']}
    if not any(m['changed'] for m in modules.values()):
        return None
    bridge = read_channel_file('server', 'server-v2.json') is None
    if bridge and not modules['server']['changed']:
        # Desktop-only batches can precede the Server bridge without breaking old users.
        bridge = False
    tag = f"server-v{modules['server']['after']}" if bridge else f'release-{identifier}'
    if tag in tags:
        raise ValueError(f'{tag} already exists; never move an immutable tag')
    return {'schema': 1, 'id': identifier, 'tag': tag, 'serverBridge': bridge, 'modules': modules,
            **({'desktopNativeSources': True} if modules['desktop']['changed'] and
               (root / 'scripts/release/native-player/windows-x64-production.json').exists() else {})}


def notes(root: Path, batch: dict) -> str:
    """生成版本快照、应用下载与单独源码下载说明，供发布前审阅。"""
    lines = [f"# Curated {batch['id']}", '', '## GitHub Release Body', '',
             'Independent Server and Desktop updates.', '', '### Module updates', '',
             '| Module | Status | Before | After |', '| --- | --- | --- | --- |']
    for component, m in batch['modules'].items():
        lines.append(f"| {component.title()} | {'Updated' if m['changed'] else 'Unchanged'} | {m['before']} | {m['after']} |")
    lines += ['', "### What's Changed", '']
    for component, m in batch['modules'].items():
        if m['changed']:
            subjects = batches.git(root, 'log', '--format=%s', f"{m['baselineCommit']}..HEAD", '--', *m['paths']).splitlines()
            lines += [f'- {component.title()}: {s}' for s in dict.fromkeys(subjects)]
    lines += ['', '### Upgrade Notes', '',
              '- Install only the components with updates. Existing library data and Desktop connections are retained.',
              '- Old all-in-one installations require the [Full 1.7.3 migration](https://github.com/yepHiu/Curated/releases/tag/full-v1.7.3) first.']
    if batch['serverBridge']:
        lines += ['- This Server release bridges older installations to the new update channel. Later updates remain available after installing it.']
    lines += ['', '### Downloads', '']
    for component, m in batch['modules'].items():
        if not m['changed']:
            lines.append(f"{component.title()} {m['after']} is unchanged; previous downloads:")
            entries = m['previousArtifacts']
        else:
            targets = [('windows', 'x64', 'exe'), ('windows', 'x64', 'zip')]
            if component == 'desktop':
                targets += [('macos', 'arm64', 'dmg'), ('macos', 'arm64', 'zip')]
            entries = []
            for platform, arch, fmt in targets:
                name = artifact_name(component, m['after'], platform, arch, fmt)
                entries.append({'fileName': name, 'url': f"https://github.com/yepHiu/Curated/releases/download/{batch['tag']}/{name}"})
        lines.extend(f"- [{a['fileName']}]({a['url']})" for a in entries)
    if batch.get('desktopNativeSources'):
        source_name = source_asset_name(batch['modules']['desktop']['after'])
        lines += ['', '### Native engine corresponding sources', '',
                  'The Desktop installer includes playback binaries and licenses. Complete corresponding '
                  'sources and build materials are available separately, at no charge, from this same Release. '
                  'They are not needed for playback or downloaded by the Desktop updater.', '',
                  f"- [{source_name}](https://github.com/yepHiu/Curated/releases/download/{batch['tag']}/{source_name})"]
    lines += ['', '### Full Changelog', '']
    for component, m in batch['modules'].items():
        if m['changed']:
            lines.append(f"- [{component.title()} changes](https://github.com/yepHiu/Curated/compare/{m['baselineCommit']}...{batch['tag']})")
    return '\n'.join(lines) + '\n'


def write_plan(root: Path, batch: dict) -> None:
    manifest = root / 'scripts/release/batches' / f"{batch['tag']}.json"
    body = root / 'docs/release-notes' / f"{batch['tag']}.md"
    if manifest.exists() or body.exists():
        raise FileExistsError('Batch/notes already exist; inspect them rather than overwrite')
    text = notes(root, batch)
    manifest.parent.mkdir(parents=True, exist_ok=True)
    body.parent.mkdir(parents=True, exist_ok=True)
    for component, module in batch['modules'].items():
        file = root / batches.VERSION_FILES[component]
        value = {'schema': 1, 'current': dict(zip(('major', 'minor', 'patch'), batches.version_tuple(module['after'])))}
        file.write_text(json.dumps(value, indent=2) + '\n', encoding='utf-8')
    manifest.write_text(json.dumps(batch, indent=2) + '\n', encoding='utf-8')
    body.write_text(text, encoding='utf-8')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--date', help='Beijing date YYYYMMDD; defaults to today')
    parser.add_argument('--write', action='store_true')
    parser.add_argument('--versions', action='store_true', help='Show current independent source versions without network requests')
    for c in batches.MODULES:
        parser.add_argument(f'--{c}-bump', choices=('patch', 'minor', 'major'), default='patch')
    args = parser.parse_args()
    if args.versions:
        print(json.dumps({c: '.'.join(str(json.loads((ROOT / p).read_text())['current'][key]) for key in ('major', 'minor', 'patch'))
                          for c, p in batches.VERSION_FILES.items()}, indent=2))
        return
    if batches.git(ROOT, 'status', '--porcelain', '--untracked-files=normal'):
        raise ValueError('Commit or isolate working changes before preparing an immutable release batch')
    batch = plan(ROOT, args.date, {c: getattr(args, c + '_bump') for c in batches.MODULES})
    print(json.dumps(batch or {'status': 'no-changes'}, indent=2))
    if args.write and batch:
        write_plan(ROOT, batch)
        print('Prepared locally. Review notes, commit, then explicitly create/push the printed tag to run CD.')


if __name__ == '__main__':
    main()
