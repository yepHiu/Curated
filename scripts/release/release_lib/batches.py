"""Immutable release batches, independent component versions and change baselines."""
from datetime import datetime, timedelta, timezone
import json
import hashlib
from pathlib import Path
import re
import subprocess

MODULES = ('server', 'desktop')
BATCH_ID = re.compile(r'(\d{8})(?:-([2-9]|[1-9]\d+))?')
SEMVER = re.compile(r'(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)')
VERSION_FILES = {'server': 'backend/internal/version/server.json',
                 'desktop': 'scripts/release/versions/desktop.json'}


def batch_key(value: str) -> tuple[str, int]:
    match = BATCH_ID.fullmatch(value)
    if not match:
        raise ValueError('Expected YYYYMMDD or YYYYMMDD-2, -3, ...')
    datetime.strptime(match[1], '%Y%m%d')
    return match[1], int(match[2] or 1)


def next_id(existing: list[str], date: str | None = None) -> str:
    date = date or datetime.now(timezone(timedelta(hours=8))).strftime('%Y%m%d')
    if not re.fullmatch(r'\d{8}', date):
        raise ValueError('Date must be YYYYMMDD without a sequence')
    batch_key(date)
    numbers = [batch_key(value)[1] for value in existing if batch_key(value)[0] == date]
    number = max(numbers, default=0) + 1
    return date if number == 1 else f'{date}-{number}'


def version_tuple(value: str) -> tuple[int, ...]:
    if not isinstance(value, str) or not SEMVER.fullmatch(value):
        raise ValueError('Expected numeric component SemVer')
    return tuple(map(int, value.split('.')))


def bump(value: str, level: str = 'patch') -> str:
    parts = list(version_tuple(value))
    index = ('major', 'minor', 'patch').index(level)
    parts[index] += 1
    parts[index + 1:] = [0] * (2 - index)
    return '.'.join(map(str, parts))


def affected(path: str) -> set[str]:
    """Conservative ownership: shared runtime/build inputs affect both deliverables."""
    name = Path(path).name
    if path in ('scripts/release/prepare_batch.py', 'scripts/release/component_cd.py',
                'scripts/release/recover_release.py', 'scripts/release/cd_release.py',
                'scripts/release/release_lib/batches.py', 'scripts/release/release_lib/latest_release.py',
                'scripts/release/release_lib/sync_notes.py', 'scripts/release/release_lib/history.py',
                'scripts/release/release_lib/versioning.py', '.gitignore', 'AGENTS.md', 'CLAUDE.md'):
        return set()
    if (path in VERSION_FILES.values() or path == 'scripts/release/version.json'
            or path.startswith(('docs/', '.cursor/', '.github/', '.agents/', 'tests/',
                                'scripts/release/tests/', 'scripts/release/batches/', 'scripts/release/versions/'))
            or name.endswith(('.md', '.test.ts', '_test.go')) or name.startswith(('test_', 'vitest.', 'playwright.'))
            or path in ('scripts/release/recovery-request.json', 'scripts/release/upgrade-baseline.json')
            or name.endswith('_smoke.py') or name.endswith('_smoke.cjs')):
        return set()
    if path.startswith(('electron/', 'src/desktop-connection/', 'scripts/release/macos/')) or path == 'vite.desktop.config.ts':
        return {'desktop'}
    if path.startswith(('backend/cmd/curated-migrate/', 'backend/internal/installmigration/')) or path in ('backend/go.mod', 'backend/go.sum'):
        return set(MODULES)  # Windows installers of both products embed the Go helper.
    if path.startswith('backend/'):
        return {'server'}
    # The standalone launcher has its own layout and connectionMessages dictionary.
    if path.startswith(('src/views/', 'src/layouts/', 'src/locales/', 'src/api/', 'src/services/', 'src/composables/', 'src/components/jav-library/')):
        return {'server'}
    if path == 'vite.config.ts' or path == 'index.html':
        return {'server'}
    return set(MODULES)


def git(root: Path, *args: str) -> str:
    return subprocess.check_output(['git', *args], cwd=root, text=True).strip()


def changed_paths(root: Path, baseline: str, ref: str = 'HEAD') -> list[str]:
    if not re.fullmatch(r'[a-f0-9]{40}', baseline):
        raise ValueError('Published component requires an exact sourceCommit baseline')
    # --no-renames keeps both paths of moves in the ownership analysis.
    return [p for p in git(root, 'diff', '--name-only', '--no-renames', '-z', baseline, ref).split('\0') if p]


def input_digest(root: Path, baseline: str, paths: list[str]) -> str:
    if not paths:
        return hashlib.sha256(b'').hexdigest()
    diff = subprocess.check_output(['git', 'diff', '--binary', '--no-ext-diff', '--no-renames', baseline, 'HEAD', '--', *paths], cwd=root)
    return hashlib.sha256(diff).hexdigest()


def load_batch(root: Path, tag: str, current: dict | None = None) -> dict | None:
    """验证批次版本与模块边界；独立原生源码只能属于更新的 Desktop。"""
    if not (re.fullmatch(r'release-[0-9-]+', tag) or re.fullmatch(r'server-v[0-9.]+', tag)):
        return None
    file = root / 'scripts/release/batches' / f'{tag}.json'
    if not file.exists():
        return None
    batch = json.loads(file.read_text(encoding='utf-8'))
    batch_key(batch['id'])
    if batch.get('schema') != 1 or batch.get('tag') != tag or set(batch['modules']) != set(MODULES):
        raise ValueError('Invalid release batch')
    bridge = batch.get('serverBridge')
    if not isinstance(bridge, bool):
        raise ValueError('Batch must declare serverBridge')
    selected = []
    for component, module in batch['modules'].items():
        before, after = version_tuple(module['before']), version_tuple(module['after'])
        valid_progression = after > before if module['changed'] else after == before
        if not isinstance(module['changed'], bool) or not valid_progression:
            raise ValueError('Only changed components may advance versions')
        if not re.fullmatch(r'[a-f0-9]{40}', module['baselineCommit']):
            raise ValueError('Invalid component baseline commit')
        if not isinstance(module['paths'], list) or not all(isinstance(p, str) for p in module['paths']):
            raise ValueError('Invalid changed paths')
        if module['changed'] != bool(module['paths']):
            raise ValueError('Component change must have deliverable evidence')
        if not re.fullmatch(r'[a-f0-9]{64}', module.get('inputDigest', '')):
            raise ValueError('Invalid component input digest')
        if current is not None and current[component] != module['after']:
            raise ValueError('Batch disagrees with component version source')
        if module['changed']:
            selected.append(component)
    if not selected:
        raise ValueError('No deliverable changes: do not create a release')
    if 'desktopNativeSources' in batch and (batch['desktopNativeSources'] is not True or 'desktop' not in selected):
        raise ValueError('Separate native sources require an updated Desktop module')
    expected_tag = f"server-v{batch['modules']['server']['after']}" if bridge else f"release-{batch['id']}"
    if tag != expected_tag or (bridge and 'server' not in selected):
        raise ValueError('Invalid batch tag or Server bridge')
    return batch


def verify_changes(root: Path, batch: dict) -> None:
    for component, module in batch['modules'].items():
        paths = sorted(p for p in changed_paths(root, module['baselineCommit']) if component in affected(p))
        if paths != sorted(module['paths']) or input_digest(root, module['baselineCommit'], paths) != module['inputDigest']:
            raise ValueError(f'{component} deliverables changed since batch preparation; prepare a new batch')


def selected_components(meta: dict) -> tuple[str, ...]:
    if 'batch' in meta:
        return tuple(c for c in MODULES if meta['batch']['modules'][c]['changed'])
    return ('server', 'desktop', 'full') if meta['component'] == 'full' else (meta['component'],)
