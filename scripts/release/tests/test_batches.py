import copy
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from scripts.release import component_cd as cd, prepare_batch
from scripts.release.release_lib import batches, component_channels
from scripts.release.release_lib.latest_release import select_latest


class BatchTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.git('init', '-q')
        self.git('config', 'user.name', 'Release tests')
        self.git('config', 'user.email', 'release@example.test')
        for c, path in batches.VERSION_FILES.items():
            self.write(path, json.dumps({'schema': 1, 'current': {'major': 1, 'minor': 0, 'patch': 0}}))
        self.write('scripts/release/versions/full.json', '{"schema":1,"current":{"major":1,"minor":0,"patch":0}}')
        self.write('backend/internal/server/example.go', 'old')
        self.write('electron/main.ts', 'old')
        self.commit()
        self.baseline = self.git('rev-parse', 'HEAD')
        self.feeds = {c: {'schema': 1, 'component': c, 'version': '1.0.0', 'sourceCommit': self.baseline, 'artifacts': []}
                      for c in batches.MODULES}
        self.addCleanup(patch.stopall)
        patch.object(prepare_batch, 'read_channel', side_effect=lambda c: self.feeds[c]).start()
        patch.object(prepare_batch, 'read_channel_file', return_value=self.feeds['server']).start()
        patch.object(cd, 'read_channel', side_effect=lambda c: self.feeds[c]).start()
        original = batches.git
        patch.object(batches, 'git', side_effect=lambda root, *args: '' if args[0] == 'ls-remote' else original(root, *args)).start()

    def git(self, *args):
        return subprocess.check_output(['git', *args], cwd=self.root, text=True).strip()

    def write(self, path, text):
        file = self.root / path
        file.parent.mkdir(parents=True, exist_ok=True)
        file.write_text(text)

    def commit(self):
        self.git('add', '.')
        self.git('commit', '-qm', 'Fix deliverable')

    def prepare(self, components=('server', 'desktop'), bridge=False):
        for c in components:
            self.write('backend/internal/server/example.go' if c == 'server' else 'electron/main.ts', 'changed')
        self.commit()
        with patch.object(prepare_batch, 'read_channel_file', return_value=None if bridge else self.feeds['server']):
            batch = prepare_batch.plan(self.root, '20261001')
        prepare_batch.write_plan(self.root, batch)
        self.commit()
        self.git('tag', batch['tag'])
        return batch, cd.metadata(self.root, batch['tag'])

    def stage(self, meta):
        windows, macos = self.root / 'windows', self.root / 'macos'
        windows.mkdir(); macos.mkdir()
        entries = []
        for name, identity in cd.expected_assets(meta).items():
            folder = windows if identity['platform'] == 'windows' else macos
            (folder / name).write_bytes(name.encode())
            entries.append({**identity, 'sourceCommit': meta['commit'], 'sha256': cd.legacy.sha256(folder / name)})
        (windows / 'windows-components.json').write_text(json.dumps({'sourceCommit': meta['commit'], 'artifacts': [e for e in entries if e['platform'] == 'windows']}))
        # Real Mac builder stores the source on its enclosing manifest, not each asset.
        (macos / 'desktop-macos.json').write_text(json.dumps({'schema': 1, 'component': 'desktop', 'version': meta['versions']['desktop'],
            'platform': 'macos', 'arch': 'arm64', 'distribution': 'desktop', 'sourceCommit': meta['commit'],
            'artifacts': [{k: e[k] for k in ('fileName', 'sha256')} for e in entries if e['platform'] == 'macos']}))
        output = self.root / 'staged'
        cd.stage(self.root, meta, windows, macos, output)
        return output

    def test_date_sequence_and_invalid_ids(self):
        self.assertEqual(batches.next_id([], '20261001'), '20261001')
        self.assertEqual(batches.next_id(['20261001', '20261001-2', '20260930-9'], '20261001'), '20261001-3')
        self.assertEqual(batches.next_id(['20261001-9'], '20261001'), '20261001-10')
        self.assertEqual(batches.next_id(['20261001-3'], '20261002'), '20261002')
        for value in ('20260230', '20261001-1', '20261001-02', '20261001123000', '../20261001'):
            with self.subTest(value=value), self.assertRaises(ValueError):
                batches.batch_key(value)

    def test_documentation_tests_and_ci_do_not_allocate(self):
        self.write('docs/guide.md', 'Docs')
        self.write('backend/internal/server/example_test.go', 'test')
        self.write('.github/workflows/ci.yml', 'ci')
        self.commit()
        self.assertIsNone(prepare_batch.plan(self.root, '20261001'))

    def test_hosted_layout_and_translations_do_not_release_desktop(self):
        self.write('src/layouts/AppShell.vue', 'Updated hosted layout')
        self.write('src/locales/en.json', '{"topics": "Updated hosted translations"}')
        self.commit()
        batch = prepare_batch.plan(self.root, '20261003')
        prepare_batch.write_plan(self.root, batch)
        self.commit()
        self.git('tag', batch['tag'])
        meta = cd.metadata(self.root, batch['tag'])
        batches.verify_changes(self.root, batch)
        self.assertEqual(meta['component'], 'server')
        self.assertEqual(meta['versions']['desktop'], '1.0.0')
        self.assertEqual({asset['component'] for asset in cd.expected_assets(meta).values()}, {'server'})
        self.assertIn('| Desktop | Unchanged | 1.0.0 | 1.0.0 |', cd.release_body(self.root, meta))

    def test_single_component_versions_and_channels_are_independent(self):
        batch, meta = self.prepare(('server',))
        self.assertEqual(meta['component'], 'server')
        self.assertEqual(meta['versions']['desktop'], '1.0.0')
        self.assertEqual(meta['versions']['server'], '1.0.1')
        batches.verify_changes(self.root, batch)
        self.assertEqual(cd.release_title(cd.release_body(self.root, meta), meta), 'Curated 20261001')
        output = self.stage(meta)
        manifest = cd.verify_distribution(meta, output / 'assets')
        self.assertEqual({e['component'] for e in manifest['artifacts']}, {'server'})
        with patch.object(cd, 'read_channel_file', return_value=self.feeds['server']):
            changes = cd.batch_channels(meta, cd.validate_channel_advance(output / 'assets'), output / 'assets')
        self.assertEqual(set(changes), {'server-v2.json'})

    def test_both_components_share_one_release_and_no_full_package(self):
        batch, meta = self.prepare()
        self.assertEqual(meta['component'], 'both')
        batches.verify_changes(self.root, batch)
        output = self.stage(meta)
        manifest = cd.verify_distribution(meta, output / 'assets')
        self.assertEqual(len(manifest['artifacts']), 6)
        self.assertEqual({e['component'] for e in manifest['artifacts']}, {'server', 'desktop'})
        self.assertTrue(all(f"/{batch['tag']}/" in e['url'] for e in manifest['artifacts']))
        with patch.object(cd, 'read_channel_file', return_value=self.feeds['server']):
            changes = cd.batch_channels(meta, cd.validate_channel_advance(output / 'assets'), output / 'assets')
        self.assertEqual(set(changes), {'server-v2.json', 'desktop.json'})
        next((output / 'assets').glob('*.dmg')).unlink()
        with self.assertRaises(ValueError):
            cd.verify_distribution(meta, output / 'assets')

    def test_desktop_only_release_does_not_require_server_bridge(self):
        batch, meta = self.prepare(('desktop',), bridge=True)
        self.assertFalse(batch['serverBridge'])
        self.assertEqual(batch['tag'], 'release-20261001')
        output = self.stage(meta)
        self.assertEqual(len(cd.verify_distribution(meta, output / 'assets')['artifacts']), 4)
        changes = cd.batch_channels(meta, cd.validate_channel_advance(output / 'assets'), output / 'assets')
        self.assertEqual(set(changes), {'desktop.json'})

    def test_bridge_updates_both_server_feeds_and_recovery_is_idempotent(self):
        batch, meta = self.prepare(bridge=True)
        self.assertEqual(batch['tag'], 'server-v1.0.1')
        self.assertEqual(cd.release_title(cd.release_body(self.root, meta), meta), 'Curated 20261001')
        output = self.stage(meta)
        changes = cd.validate_channel_advance(output / 'assets')
        with patch.object(cd, 'read_channel_file', return_value=None):
            changes = cd.batch_channels(meta, changes, output / 'assets')
        self.assertEqual(set(changes), {'server.json', 'server-v2.json', 'desktop.json'})
        self.assertEqual(changes['server.json'], changes['server-v2.json'])
        new = json.loads(changes['server.json'])
        with patch.object(cd, 'read_channel_file', return_value=new):
            self.assertEqual(cd.batch_channels(meta, {}, output / 'assets'), {})
        with patch.object(cd, 'read_channel_file', return_value=self.feeds['server']), self.assertRaisesRegex(ValueError, 'already exists'):
            cd.batch_channels(meta, changes, output / 'assets')

    def test_date_tagged_server_cannot_skip_bridge(self):
        _, meta = self.prepare(('server',))
        output = self.stage(meta)
        with patch.object(cd, 'read_channel_file', return_value=None), self.assertRaisesRegex(ValueError, 'bridge'):
            cd.batch_channels(meta, cd.validate_channel_advance(output / 'assets'), output / 'assets')

    def test_new_deliverable_or_channel_baseline_invalidates_prepared_batch(self):
        batch, meta = self.prepare(('server',))
        self.write('electron/main.ts', 'later change')
        self.commit()
        with self.assertRaisesRegex(ValueError, 'since batch preparation'):
            batches.verify_changes(self.root, batch)
        self.feeds['server']['sourceCommit'] = 'b' * 40
        with self.assertRaisesRegex(ValueError, 'baseline changed'):
            cd.release_body(self.root, meta)

    def test_edits_to_already_selected_file_invalidate_input_digest(self):
        batch, _ = self.prepare(('server',))
        self.write('backend/internal/server/example.go', 'another implementation')
        self.commit()
        with self.assertRaisesRegex(ValueError, 'since batch preparation'):
            batches.verify_changes(self.root, batch)

    def test_tampered_artifact_source_or_release_url_is_rejected(self):
        _, meta = self.prepare(('server',))
        output = self.stage(meta)
        manifest = cd.verify_distribution(meta, output / 'assets')
        for key, value in (('sourceCommit', 'b' * 40), ('url', 'https://github.com/yepHiu/Curated/releases/download/release-20260930/old.exe')):
            entries = copy.deepcopy(manifest['artifacts'])
            entries[0][key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                cd.validate_entries(meta, entries, [output / 'assets'])

    def test_ownership_accounts_for_hosted_web_and_shared_installer(self):
        self.assertEqual(batches.affected('src/views/Library.vue'), {'server'})
        self.assertEqual(batches.affected('src/desktop-connection/ConnectionPage.vue'), {'desktop'})
        for path in ('scripts/release/windows/Component.iss.tpl', 'backend/internal/installmigration/stop_windows.go',
                     'pnpm-lock.yaml', 'src/components/ui/button/Button.vue', 'src/lib/theme-storage.ts'):
            self.assertEqual(batches.affected(path), {'server', 'desktop'})

    def test_feed_reader_uses_new_server_channel_and_only_falls_back_on_absence(self):
        with patch.object(component_channels, 'read_channel_file', side_effect=[None, self.feeds['server']]) as read:
            self.assertEqual(component_channels.read_channel('server'), self.feeds['server'])
            self.assertEqual([c.args for c in read.call_args_list], [('server', 'server-v2.json'), ('server', 'server.json')])
        with patch.object(component_channels, 'read_channel_file', side_effect=ValueError('invalid')), self.assertRaises(ValueError):
            component_channels.read_channel('server')

    def test_latest_selects_batch_sequence_numerically_including_bridge(self):
        def release(identifier, tag=None, **extra):
            return {'tag_name': tag or f'release-{identifier}', 'body': f'<!-- curated-release-batch:{identifier} -->',
                    'draft': False, 'prerelease': False, 'assets': [{'name': n, 'size': 10} for n in
                    ('release.json', 'SHA256SUMS.txt', 'Curated-Server-Setup-1.0.1-windows-x64.exe')], **extra}
        bridge = release('20261001', 'server-v1.0.1')
        latest = release('20261001-10')
        self.assertEqual(select_latest([bridge, release('20261001-9'), latest, release('20261002', draft=True)]), latest)
        self.assertEqual(select_latest([bridge]), bridge)
        with self.assertRaisesRegex(ValueError, 'missing'):
            select_latest([release('20261001', assets=[])])


if __name__ == '__main__':
    unittest.main()
