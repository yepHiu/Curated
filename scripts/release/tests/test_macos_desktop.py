import json
from pathlib import Path
import plistlib
import tempfile
import unittest
from unittest.mock import patch

from scripts.release.release_lib.macos_desktop import stage_app, mac_artifact_names, package_macos_desktop
from scripts.release import cd_release as cd


class MacDesktopTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)

    def write(self, name, data=b'fixture'):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)
        return path

    def test_standalone_bundle_contains_only_client_assets(self):
        runtime = 'node_modules/electron/dist/'
        binary = self.write(runtime + 'Electron.app/Contents/MacOS/Electron')
        binary.chmod(0o755)
        self.write(runtime + 'Electron.app/Contents/Info.plist', plistlib.dumps({'CFBundleIdentifier': 'com.github.Electron'}))
        self.write(runtime + 'Electron.app/Contents/Resources/default_app.asar')
        self.write(runtime + 'LICENSE')
        self.write(runtime + 'LICENSES.chromium.html')
        for name in ('launcher/index.html', 'launcher-preload.cjs', 'main.js', 'preload.cjs', 'connections.html', 'connections-preload.cjs', 'connections-ui.js', 'connections.css', 'connections-tokens.css'):
            self.write('electron-dist/' + name)
        self.write('electron-dist/main.js.map')
        self.write('electron-dist/desktop-release.json', b'{"distribution":"legacy"}')
        self.write('public/Curated-icon.png')
        self.write('public/Curated-icon-macos.png')
        self.write('LICENSE')
        self.write('backend/runtime/curated.db')
        self.write('frontend-dist/index.html')
        self.write('backend/third_party/ffmpeg/bin/ffmpeg')
        app = self.root / 'out/Curated Desktop.app'
        stage_app(self.root, app, '0.1.0', '20260927.000000')
        payload = app / 'Contents/Resources/app'
        metadata = json.loads((payload / 'electron-dist/desktop-release.json').read_text())
        self.assertEqual(metadata['distribution'], 'desktop')
        self.assertEqual(metadata['updateFeed'], 'https://raw.githubusercontent.com/yepHiu/Curated/release-channels/desktop.json')
        self.assertEqual(metadata['version'], '0.1.0')
        self.assertEqual(json.loads((self.root / 'electron-dist/desktop-release.json').read_text())['distribution'], 'legacy')
        self.assertEqual({p.name for p in payload.iterdir()}, {'package.json', 'electron-dist', 'public'})
        self.assertFalse((payload / 'electron-dist/main.js.map').exists())
        self.assertFalse((app / 'Contents/Resources/default_app.asar').exists())
        self.assertEqual((app / 'Contents/MacOS/Curated Desktop').stat().st_mode & 0o777, 0o755)
        with (app / 'Contents/Info.plist').open('rb') as stream:
            info = plistlib.load(stream)
        self.assertEqual(info['CFBundleIdentifier'], 'com.curated.desktop')
        self.assertEqual(info['CFBundleExecutable'], 'Curated Desktop')
        with self.assertRaises(FileExistsError):
            stage_app(self.root, app, '0.1.0', '20260927.000000')

    def test_rejects_intel_or_non_macos_hosts(self):
        for system, machine in [('Darwin', 'x86_64'), ('Linux', 'arm64')]:
            with patch('platform.system', return_value=system), patch('platform.machine', return_value=machine), self.assertRaises(RuntimeError):
                package_macos_desktop(self.root, self.root / 'out')

    def test_existing_production_package_is_not_overwritten(self):
        self.write('scripts/release/versions/desktop.json', b'{"schema":1,"current":{"major":0,"minor":1,"patch":0}}')
        original = self.write('out/' + mac_artifact_names('0.1.0')[0], b'keep existing package')
        with patch('platform.system', return_value='Darwin'), patch('platform.machine', return_value='arm64'), self.assertRaises(FileExistsError):
            package_macos_desktop(self.root, self.root / 'out')
        self.assertEqual(original.read_bytes(), b'keep existing package')

    def mac_fixture(self):
        self.write('scripts/release/versions/desktop.json', b'{"schema":1,"current":{"major":0,"minor":1,"patch":0}}')
        manifest = {'schema':1, 'component':'desktop', 'version':'0.1.0', 'platform':'macos', 'arch':'arm64', 'distribution':'desktop', 'sourceCommit':'abc', 'artifacts':[]}
        for name in mac_artifact_names('0.1.0'):
            path = self.write('mac/' + name)
            manifest['artifacts'].append({'fileName':name, 'sha256':cd.sha256(path)})
        self.write('mac/desktop-macos.json', json.dumps(manifest).encode())
        metadata = {'version':'1.5.9', 'tag':'v1.5.9', 'commit':'abc'}
        for name in cd.artifact_names('1.5.9'):
            self.write('out/assets/' + name)
        self.write('out/assets/release.json', json.dumps({'version':'1.5.9', 'sourceCommit':'abc'}).encode())
        assets = self.root / 'out/assets'
        self.write('out/assets/SHA256SUMS.txt', ''.join(f'{cd.sha256(assets / name)}  {name}\n' for name in cd.artifact_names('1.5.9')).encode())
        return manifest, metadata

    def test_merges_both_platforms_and_generates_combined_checksums(self):
        _, metadata = self.mac_fixture()
        result = cd.merge_macos(self.root, self.root / 'out', self.root / 'mac', metadata)
        cd.verify_staged(self.root / 'out', result)
        self.assertEqual(result['desktopVersion'], '0.1.0')
        assets = self.root / 'out/assets'
        self.assertEqual(len(list(assets.iterdir())), 7)
        checksums = (assets / 'SHA256SUMS.txt').read_text()
        for name in [*cd.artifact_names('1.5.9'), *mac_artifact_names('0.1.0'), 'desktop-macos.json']:
            self.assertIn(f'{cd.sha256(assets / name)}  {name}', checksums)

    def test_mismatched_commit_architecture_and_corrupt_packages_block_publication(self):
        manifest, metadata = self.mac_fixture()
        for field, value in [('sourceCommit','wrong'), ('arch','x64'), ('version','0.2.0')]:
            bad = {**manifest, field:value}
            self.write('mac/desktop-macos.json', json.dumps(bad).encode())
            with self.assertRaises(ValueError):
                cd.merge_macos(self.root, self.root / 'out', self.root / 'mac', metadata)
        self.write('mac/desktop-macos.json', json.dumps(manifest).encode())
        self.write('mac/' + mac_artifact_names('0.1.0')[0], b'corruption')
        with self.assertRaisesRegex(ValueError, 'checksum'):
            cd.merge_macos(self.root, self.root / 'out', self.root / 'mac', metadata)
