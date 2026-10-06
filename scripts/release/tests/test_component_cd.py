import copy
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from scripts.release import component_cd as cd
from scripts.release.release_lib.windows_components import validate_payload, stage_desktop


class ComponentReleaseTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.meta = {'component': 'full', 'tag': 'full-v1.6.0', 'version': '1.6.0', 'commit': 'abc',
                     'versions': {'full': '1.6.0', 'server': '1.5.8', 'desktop': '0.1.0'}}

    def fixture(self):
        entries = []
        for name, identity in cd.expected_assets(self.meta).items():
            path = self.root / name
            path.write_bytes(name.encode())
            entry = {**identity, 'sha256': cd.legacy.sha256(path),
                     'url': f'https://github.com/yepHiu/Curated/releases/download/full-v1.6.0/{name}'}
            if entry['component'] == 'full':
                entry['components'] = {'server': '1.5.8', 'desktop': '0.1.0'}
            entries.append(entry)
        return entries

    def test_new_full_commands_fail_before_git_or_build(self):
        for command in ('check', 'windows', 'macos', 'stage', 'publish'):
            with self.subTest(command=command), \
                 patch('sys.argv', ['component_cd', command, '--tag', 'full-v1.7.4']), \
                 patch.object(cd, 'metadata') as metadata, \
                 self.assertRaisesRegex(ValueError, 'Full is retired'):
                cd.main()
            metadata.assert_not_called()

    def notes_fixture(self, component='server'):
        meta = {'component': component, 'tag': f'{component}-v1.1.0',
                'version': '1.1.0', 'commit': 'abc'}
        directory = self.root / 'docs/release-notes'
        directory.mkdir(parents=True, exist_ok=True)
        file = directory / f"{meta['tag']}.md"
        rows = [f'| {c.title()} | {"Updated" if c == component else "Unchanged"} | 1.0.0 | {"1.1.0" if c == component else "1.0.0"} |'
                for c in ('desktop', 'server')]
        text = '## GitHub Release Body\n\n### Module updates\n\n| Module | Status | Before | After |\n| --- | --- | --- | --- |\n' + '\n'.join(rows) + '\n\n### What\'s Changed\n\nFix component startup.\n'
        file.write_text(text, encoding='utf-8')
        return meta, file, text

    def test_notes_identify_both_modules_for_either_standalone_release(self):
        for component in ('server', 'desktop'):
            meta, _, text = self.notes_fixture(component)
            with self.subTest(component=component), patch.object(cd, 'read_channel', return_value={'version': '1.0.0'}):
                self.assertEqual(cd.release_body(self.root, meta), text.split('## GitHub Release Body', 1)[1].strip() + '\n\n' + cd.legacy.source_marker(meta) + '\n')

    def test_notes_reject_missing_status_rows_version_bumps_and_stale_baseline(self):
        meta, file, text = self.notes_fixture()
        invalid = [
            text.replace('| Desktop | Unchanged | 1.0.0 | 1.0.0 |', ''),
            text + '\n| Desktop | Unchanged | 1.0.0 | 1.0.0 |\n',
            text.replace('| Desktop | Unchanged | 1.0.0 | 1.0.0 |', '| Desktop | Unchanged | 1.0.0 | 1.1.0 |'),
            text.replace('| Desktop | Unchanged', '| Desktop | Updated'),
            text.replace('| Server | Updated', '| Server | Unchanged'),
            text.replace('| Server | Updated | 1.0.0 | 1.1.0 |', '| Server | Updated | 1.1.0 | 1.1.0 |'),
            text.replace('| Server | Updated | 1.0.0 | 1.1.0 |', '| Server | Updated | 1.0.0 | 1.2.0 |'),
            text.replace('1.0.0', '0.9.0'),
        ]
        for notes in invalid:
            file.write_text(notes, encoding='utf-8')
            with self.subTest(notes=notes), patch.object(cd, 'read_channel', return_value={'version': '1.0.0'}), self.assertRaises(ValueError):
                cd.release_body(self.root, meta)

    def test_title_uses_note_snapshot_without_live_channels_or_source_versions(self):
        for component in ('server', 'desktop'):
            meta, _, text = self.notes_fixture(component)
            meta['versions'] = {'server': '99.0.0', 'desktop': '99.0.0'}
            expected = ('Server 1.1.0 + Desktop 1.0.0' if component == 'server'
                        else 'Server 1.0.0 + Desktop 1.1.0')
            with self.subTest(component=component), patch.object(cd, 'read_channel') as channel:
                self.assertEqual(cd.release_title(text, meta),
                                 f'Curated - {expected} - {component.title()} update')
                channel.assert_not_called()
            with self.assertRaises(ValueError):
                cd.release_title(text.replace('| Unchanged |', '| Updated |'), meta)

    def test_historical_full_title_is_preserved(self):
        self.assertEqual(cd.release_title('Historical notes', self.meta), 'Curated v1.6.0')

    def test_invalid_notes_stop_publication_before_upload(self):
        meta, file, _ = self.notes_fixture()
        file.write_text('## GitHub Release Body\nUndeclared update', encoding='utf-8')
        with patch.object(cd.legacy, 'api') as api, patch.object(cd.subprocess, 'run') as run, self.assertRaises(ValueError):
            cd.publish(self.root, meta, self.root, 'publish')
        api.assert_not_called()
        run.assert_not_called()

    def test_other_module_release_does_not_invalidate_unchanged_snapshot(self):
        meta, file, text = self.notes_fixture()
        with patch.object(cd, 'read_channel', side_effect=lambda c: {'version': '1.1.0' if c == 'desktop' else '1.0.0'}):
            cd.release_body(self.root, meta)
        file.write_text(text.replace('| Desktop | Unchanged | 1.0.0 | 1.0.0 |',
                                     '| Desktop | Unchanged | 1.2.0 | 1.2.0 |'), encoding='utf-8')
        with patch.object(cd, 'read_channel', return_value={'version': '1.0.0'}), self.assertRaisesRegex(ValueError, 'unpublished'):
            cd.release_body(self.root, meta)

    def test_full_requires_all_eight_packages_and_actual_component_versions(self):
        entries = self.fixture()
        self.assertEqual(len(entries), 8)
        cd.validate_entries(self.meta, entries, [self.root])
        for bad in (entries[:-1], [entries[0], *entries]):
            with self.assertRaises(ValueError):
                cd.validate_entries(self.meta, bad, [self.root])
        bad = copy.deepcopy(entries)
        next(e for e in bad if e['component'] == 'full')['components']['server'] = '1.6.0'
        with self.assertRaises(ValueError):
            cd.validate_entries(self.meta, bad, [self.root])

    def test_stage_and_verify_combined_distribution(self):
        entries = self.fixture()
        windows = self.root / 'windows'
        macos = self.root / 'macos'
        windows.mkdir(); macos.mkdir()
        for entry in entries:
            directory = windows if entry['platform'] == 'windows' else macos
            (self.root / entry['fileName']).rename(directory / entry['fileName'])
        (windows / 'windows-components.json').write_text(json.dumps({'sourceCommit': 'abc',
            'artifacts': [e for e in entries if e['platform'] == 'windows']}))
        (macos / 'desktop-macos.json').write_text(json.dumps({'schema': 1, 'component': 'desktop',
            'version': '0.1.0', 'platform': 'macos', 'arch': 'arm64', 'distribution': 'desktop',
            'sourceCommit': 'abc', 'artifacts': [e for e in entries if e['platform'] == 'macos']}))
        output = self.root / 'output'
        cd.stage(self.root, self.meta, windows, macos, output)
        manifest = cd.verify_distribution(self.meta, output / 'assets')
        self.assertEqual(len(manifest['artifacts']), 8)
        desktop = output / 'assets/desktop.json'
        desktop.write_text('{}')
        with self.assertRaises(ValueError):
            cd.verify_distribution(self.meta, output / 'assets')

    def test_wrong_component_arch_and_corrupt_binary_fail(self):
        entries = self.fixture()
        for key, value in [('component', 'desktop'), ('arch', 'arm64'), ('version', '1.6.0'), ('sha256', '0'*64)]:
            bad = copy.deepcopy(entries)
            bad[0][key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                cd.validate_entries(self.meta, bad, [self.root])

    def test_channel_cannot_replace_published_component_bytes_or_downgrade(self):
        entries = self.fixture()
        document = {'schema': 1, 'component': 'server', 'version': '1.5.8',
                    'artifacts': [e for e in entries if e['component'] == 'server']}
        file = self.root / 'server.json'
        file.write_text(json.dumps(document))
        with patch.object(cd, 'read_channel', return_value=document):
            self.assertEqual(cd.validate_channel_advance(self.root), {})
            bad = copy.deepcopy(document)
            bad['artifacts'][0]['sha256'] = '0'*64
            file.write_text(json.dumps(bad))
            with self.assertRaises(ValueError):
                cd.validate_channel_advance(self.root)
            bad['version'] = '1.5.7'
            file.write_text(json.dumps(bad))
            with self.assertRaises(ValueError):
                cd.validate_channel_advance(self.root)

    def test_publish_updates_channels_then_reconciles_latest(self):
        entries = self.fixture()
        assets = self.root / 'output/assets'
        assets.mkdir(parents=True)
        for entry in entries:
            (self.root / entry['fileName']).rename(assets / entry['fileName'])
        (assets / 'release.json').write_text(json.dumps({**self.meta, 'artifacts': entries}))
        release = {'id': 20, 'html_url': 'https://example.test/release'}
        with patch.object(cd.legacy, 'check_release', return_value=None), patch.object(cd, 'release_body', return_value='Notes'), \
             patch.object(cd, 'reconcile_latest') as reconcile, patch.object(cd.legacy, 'api', return_value=release) as api, \
             patch.object(cd.legacy, 'verify_uploaded'), patch.object(cd.subprocess, 'run'), \
             patch.object(cd, 'verify_distribution'), patch.object(cd, 'validate_channel_advance', return_value={}), patch.object(cd, 'advance_channels'), \
             patch.dict('os.environ', {'GITHUB_REPOSITORY': 'yepHiu/Curated'}):
            cd.publish(self.root, self.meta, self.root / 'output', 'publish')
            payloads = [call.args for call in api.call_args_list if len(call.args) > 1]
            self.assertNotIn('target_commitish', next(args[1] for args in payloads if args[0] == 'releases'))
            reconcile.assert_called_once_with(cd.legacy.api)
            self.assertIn(('releases/20', {'draft': False, 'make_latest': 'false'}, 'PATCH'), payloads)
            self.assertFalse(any(args[0] == 'releases/20' and args[1].get('make_latest') == 'true' for args in payloads))

    def test_preflight_never_requires_or_mutates_latest(self):
        self.meta.update(component='server', tag='server-v1.5.8', version='1.5.8')
        self.meta['batch'] = {'id': '20261001'}
        with patch('sys.argv', ['component_cd', 'check', '--tag', self.meta['tag']]), \
             patch.object(cd, 'metadata', return_value=self.meta), \
             patch.object(cd, 'verify_changes'), \
             patch.object(cd, 'check_batch_identity'), \
             patch.object(cd, 'release_body', return_value='Notes'), \
             patch.object(cd.legacy, 'check_release') as check, \
             patch.object(cd.legacy, 'api') as api, \
             patch.object(cd, 'reconcile_latest') as reconcile:
            cd.main()
            check.assert_called_once_with(self.meta)
            api.assert_not_called()
            reconcile.assert_not_called()

    def test_public_release_recovery_advances_channels_before_latest(self):
        calls = []
        release = {'draft': False, 'body': cd.legacy.source_marker(self.meta)}
        with patch('sys.argv', ['component_cd', 'channels', '--tag', self.meta['tag']]), \
             patch.object(cd, 'metadata', return_value=self.meta), \
             patch.object(cd.legacy, 'find_release', return_value=release), \
             patch.object(cd, 'verify_distribution'), patch.object(cd.legacy, 'verify_uploaded'), \
             patch.object(cd, 'validate_channel_advance', return_value={}), \
             patch.object(cd, 'advance_channels', side_effect=lambda *_: calls.append('channels')), \
             patch.object(cd, 'reconcile_latest', side_effect=lambda *_: calls.append('latest')):
            cd.main()
        self.assertEqual(calls, ['channels', 'latest'])

    def test_draft_does_not_change_channels_or_latest(self):
        assets = self.root / 'assets'
        assets.mkdir()
        with patch.object(cd.legacy, 'check_release', return_value=None), patch.object(cd, 'release_body', return_value='Notes'), \
             patch.object(cd.legacy, 'api', return_value={'id': 20, 'html_url': 'fixture'}) as api, \
             patch.object(cd.legacy, 'verify_uploaded'), patch.object(cd.subprocess, 'run'), \
             patch.object(cd, 'verify_distribution'), patch.object(cd, 'validate_channel_advance', return_value={}), \
             patch.object(cd, 'advance_channels') as advance, patch.object(cd, 'reconcile_latest') as reconcile, \
             patch.dict('os.environ', {'GITHUB_REPOSITORY': 'yepHiu/Curated'}):
            cd.publish(self.root, self.meta, self.root, 'draft')
            advance.assert_not_called()
            reconcile.assert_not_called()
            payloads = [call.args[1] for call in api.call_args_list if len(call.args) > 1]
            self.assertTrue(all(p.get('draft') is True and p.get('make_latest') == 'false' for p in payloads))

    def test_initial_channel_branch_contains_only_manifests_without_source_history(self):
        def response(endpoint, payload=None, method=None):
            if endpoint == 'git/matching-refs/heads/release-channels': return []
            if endpoint == 'git/trees':
                self.assertNotIn('base_tree', payload)
                self.assertEqual([e['path'] for e in payload['tree']], ['server.json'])
                return {'sha': 'tree'}
            if endpoint == 'git/commits':
                self.assertEqual(payload['parents'], [])
                return {'sha': 'channel'}
            if endpoint == 'git/refs':
                self.assertEqual(payload, {'ref': 'refs/heads/release-channels', 'sha': 'channel'})
                return {}
            self.fail(endpoint)
        with patch.object(cd.legacy, 'api', side_effect=response):
            cd.advance_channels({'server.json': '{}'}, self.meta)

    def test_server_payload_cannot_contain_electron(self):
        for name in ('curated.exe', 'frontend-dist/index.html', 'third_party/ffmpeg/bin/ffmpeg.exe', 'third_party/ffmpeg/bin/ffprobe.exe'):
            file = self.root / name
            file.parent.mkdir(parents=True, exist_ok=True)
            file.write_text('fixture')
        validate_payload(self.root, 'server')
        (self.root / 'electron.exe').write_text('bad')
        with self.assertRaises(ValueError):
            validate_payload(self.root, 'server')

    def test_desktop_stages_only_client_runtime(self):
        for name in ('node_modules/electron/dist/electron.exe', 'electron-dist/main.js', 'electron-dist/desktop-release.json',
                     'electron-dist/native-player-host.exe', 'electron-dist/player/index.html', 'electron-dist/playback-preload.cjs',
                     'public/Curated-desktop-icon.png', 'icon/curated-desktop.ico', 'LICENSE', 'backend/curated.exe'):
            file = self.root / name
            file.parent.mkdir(parents=True, exist_ok=True)
            file.write_text('fixture')
        server_icon = self.root / 'backend/internal/assets/curated.ico'
        server_icon.parent.mkdir(parents=True, exist_ok=True)
        server_icon.write_text('dark server icon')
        from scripts.release.tests.test_native_player import NativeEnginePayloadTests
        bundle = self.root / 'engine'
        bundle.mkdir()
        NativeEnginePayloadTests().fixture(bundle)
        from scripts.release.release_lib.native_player import stage_native_sources
        source_asset = stage_native_sources(bundle, self.root / 'source-release', '0.1.0', 'release-20261007-2')
        with patch.dict('os.environ', {'CURATED_NATIVE_BUNDLE': str(bundle)}):
            stage_desktop(self.root, self.root / 'desktop', '0.1.0', '20260927.000000', source_asset=source_asset)
        app = self.root / 'desktop/resources/app'
        self.assertEqual(json.loads((app / 'package.json').read_text())['version'], '0.1.0')
        self.assertEqual((self.root / 'electron-dist/desktop-release.json').read_text(), 'fixture')
        self.assertEqual((self.root / 'desktop/curated.ico').read_text(), 'fixture')
        self.assertEqual((app / 'curated-desktop.ico').read_text(), 'fixture')
        self.assertEqual((app / 'public/Curated-desktop-icon.png').read_text(), 'fixture')
        self.assertFalse((app / 'public/Curated-icon.png').exists())
        self.assertFalse((app / 'native-player/sources/complete.tar.zst').exists())
        (app / 'curated.exe').write_text('bad')
        with self.assertRaises(ValueError):
            validate_payload(self.root / 'desktop', 'desktop')

    def test_separate_sources_are_required_verified_and_excluded_from_update_feeds(self):
        """新批次必须上传完整来源资产，但 Desktop 更新清单只列应用安装包。"""
        from scripts.release.tests.test_native_player import NativeEnginePayloadTests
        from scripts.release.release_lib.native_player import stage_native_sources
        self.meta.update(component='both', tag='release-20261007-2',
                         batch={'desktopNativeSources': True, 'modules': {
                             'server': {'changed': True}, 'desktop': {'changed': True}}})
        windows, macos = self.root / 'windows', self.root / 'macos'
        windows.mkdir(); macos.mkdir()
        entries = []
        for name, identity in cd.expected_assets(self.meta).items():
            folder = windows if identity['platform'] == 'windows' else macos
            (folder / name).write_bytes(name.encode())
            entries.append({**identity, 'sourceCommit': 'abc', 'sha256': cd.legacy.sha256(folder / name)})
        lock = self.root / 'scripts/release/native-player/windows-x64-production.json'
        lock.parent.mkdir(parents=True)
        # Model the Windows producer's autocrlf checkout and the Linux publisher.
        lock.write_bytes(b'{\r\n}\r\n')
        bundle = self.root / 'engine'
        bundle.mkdir()
        manifest = NativeEnginePayloadTests().fixture(bundle)
        inputs = bundle / 'sources/build-inputs.json'
        inputs.write_text('{"lock":{}}')
        manifest.update(lockSha256=cd.legacy.sha256(lock))
        manifest['buildInputs']['sha256'] = cd.legacy.sha256(inputs)
        (bundle / 'engine-manifest.json').write_text(json.dumps(manifest))
        source = stage_native_sources(bundle, windows, '0.1.0', self.meta['tag'])
        source['sourceCommit'] = 'abc'
        lock.write_bytes(b'{\n}\n')
        with self.assertRaisesRegex(ValueError, 'requires a separate'):
            cd.validate_native_sources(self.root, self.meta, None, [windows])
        cd.validate_native_sources(self.root, self.meta, source, [windows])
        lock.write_bytes(b'{"changed":true}\n')
        with self.assertRaisesRegex(ValueError, 'frozen release engine lock'):
            cd.validate_native_sources(self.root, self.meta, source, [windows])
        lock.write_bytes(b'{\n}\n')
        wrong = {**source, 'sourceCommit': 'another'}
        with self.assertRaisesRegex(ValueError, 'another Desktop'):
            cd.validate_native_sources(self.root, self.meta, wrong, [windows])
        (windows / 'windows-components.json').write_text(json.dumps({'sourceCommit': 'abc',
            'artifacts': [e for e in entries if e['platform'] == 'windows'], 'nativeSources': source}))
        (macos / 'desktop-macos.json').write_text(json.dumps({'schema': 1, 'component': 'desktop',
            'version': '0.1.0', 'platform': 'macos', 'arch': 'arm64', 'distribution': 'desktop',
            'sourceCommit': 'abc', 'artifacts': [e for e in entries if e['platform'] == 'macos']}))
        output = self.root / 'output'
        cd.stage(self.root, self.meta, windows, macos, output)
        verified = cd.verify_distribution(self.meta, output / 'assets', self.root)
        self.assertEqual(verified['nativeSources'], source)
        feed = json.loads((output / 'assets/desktop.json').read_text())
        self.assertNotIn(source['fileName'], {e['fileName'] for e in feed['artifacts']})
        self.assertIn(source['fileName'], (output / 'assets/SHA256SUMS.txt').read_text())
        (output / 'assets' / source['fileName']).write_bytes(b'corrupt')
        with self.assertRaisesRegex(ValueError, 'checksum'):
            cd.verify_distribution(self.meta, output / 'assets', self.root)
