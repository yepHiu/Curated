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

    def test_wrong_component_arch_and_corrupt_binary_fail(self):
        entries = self.fixture()
        for key, value in [('component', 'desktop'), ('arch', 'arm64'), ('version', '1.6.0'), ('sha256', '0'*64)]:
            bad = copy.deepcopy(entries)
            bad[0][key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                cd.validate_entries(self.meta, bad, [self.root])

    def test_old_updater_feed_accepts_only_legacy_setup(self):
        release = {'id': 1, 'tag_name': 'v1.5.8', 'assets': [{'name': 'Curated-Setup-1.5.8.exe'}]}
        with patch.object(cd.legacy, 'api', return_value=release):
            self.assertEqual(cd.legacy_latest(), release)
            release['assets'].append({'name': 'Curated-Desktop-Setup-0.1.0-windows-x64.exe'})
            with self.assertRaises(ValueError):
                cd.legacy_latest()
            release['tag_name'] = 'full-v1.6.0'
            with self.assertRaises(ValueError):
                cd.legacy_latest()

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

    def test_publish_pins_legacy_and_does_not_promote_split_to_latest(self):
        entries = self.fixture()
        assets = self.root / 'output/assets'
        assets.mkdir(parents=True)
        for entry in entries:
            (self.root / entry['fileName']).rename(assets / entry['fileName'])
        (assets / 'release.json').write_text(json.dumps({**self.meta, 'artifacts': entries}))
        latest = {'id': 10}
        release = {'id': 20, 'html_url': 'https://example.test/release'}
        with patch.object(cd.legacy, 'check_release', return_value=None), patch.object(cd, 'body', return_value='Notes'), \
             patch.object(cd, 'legacy_latest', return_value=latest), patch.object(cd.legacy, 'api', return_value=release) as api, \
             patch.object(cd.legacy, 'verify_uploaded'), patch.object(cd.subprocess, 'run'), \
             patch.object(cd, 'verify_distribution'), patch.object(cd, 'validate_channel_advance', return_value={}), patch.object(cd, 'advance_channels'), \
             patch.dict('os.environ', {'GITHUB_REPOSITORY': 'yepHiu/Curated'}):
            cd.publish(self.root, self.meta, self.root / 'output', 'publish')
            payloads = [call.args for call in api.call_args_list if len(call.args) > 1]
            self.assertIn(('releases/10', {'make_latest': 'true'}, 'PATCH'), payloads)
            self.assertIn(('releases/20', {'draft': False, 'make_latest': 'false'}, 'PATCH'), payloads)
            self.assertFalse(any(args[0] == 'releases/20' and args[1].get('make_latest') == 'true' for args in payloads))

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
                     'public/Curated-icon.png', 'backend/internal/assets/curated.ico', 'LICENSE', 'backend/curated.exe'):
            file = self.root / name
            file.parent.mkdir(parents=True, exist_ok=True)
            file.write_text('fixture')
        stage_desktop(self.root, self.root / 'desktop', '0.1.0', '20260927.000000')
        app = self.root / 'desktop/resources/app'
        self.assertEqual(json.loads((app / 'package.json').read_text())['version'], '0.1.0')
        self.assertEqual((self.root / 'electron-dist/desktop-release.json').read_text(), 'fixture')
        (app / 'curated.exe').write_text('bad')
        with self.assertRaises(ValueError):
            validate_payload(self.root / 'desktop', 'desktop')
