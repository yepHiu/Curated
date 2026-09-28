"""Real Windows installer lifecycle tests with disposable identities and data."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time
import unittest

from scripts.release.release_lib.windows_components import _find_iscc


@unittest.skipUnless(sys.platform == 'win32' and _find_iscc(), 'Requires Windows and Inno Setup')
class InstallerFlowTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        """Compile shared disposable installers once for both lifecycle scenarios."""
        from scripts.release.tests.installer_fixture import build_fixture
        cls.temporary = tempfile.TemporaryDirectory(prefix='curated-wizard-test-')
        cls.addClassCleanup(cls.temporary.cleanup)
        cls.fixture = build_fixture(Path(cls.temporary.name) / 'fixtures')

    def installed(self, component):
        """Read only this fixture identity to observe installation version transitions."""
        import winreg
        key = 'Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\' + self.fixture['identity'] + '.' + component + '_is1'
        try:
            with winreg.OpenKey(winreg.HKEY_CURRENT_USER, key) as entry:
                return winreg.QueryValueEx(entry, 'DisplayVersion')[0]
        except FileNotFoundError:
            return None

    def install(self, component, version, target, *args):
        """Run unattended setup against the selected test directory with a unique log."""
        log = self.fixture['base'] / f'{component}-{time.time_ns()}.log'
        result = subprocess.run([str(self.fixture['setups'][(component, version)]), '/VERYSILENT',
            '/SUPPRESSMSGBOXES', '/SP-', '/NORESTART', '/NOLAUNCH=1', '/LANG=english',
            '/DIR=' + str(target), '/LOG=' + str(log), *args], timeout=60)
        return result.returncode

    def cleanup_install(self, target):
        """Uninstall a verified fixture path once and preserve all production installs."""
        if target in getattr(self, '_cleaned', set()):
            return
        self.assertTrue(target.resolve().is_relative_to(self.fixture['base']))
        uninstaller = target / 'unins000.exe'
        if uninstaller.exists():
            subprocess.run([str(uninstaller), '/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART'], timeout=30, check=True)
            self._cleaned = getattr(self, '_cleaned', set()) | {target}

    def start_fixture(self, target, component):
        """Start a hidden app that handles real Windows shutdown requests."""
        ready = target.parent / f'ready-{time.time_ns()}'
        exe = 'curated.exe' if component == 'server' else 'Curated Desktop.exe'
        startup = subprocess.STARTUPINFO()
        startup.dwFlags |= subprocess.STARTF_USESHOWWINDOW
        startup.wShowWindow = 0
        proc = subprocess.Popen([str(target / exe), '-test.run=^TestStopWindowFixture$'],
            env={**os.environ, 'CURATED_STOP_TEST_READY': str(ready)}, startupinfo=startup)
        def cleanup():
            """Stop only this test child if a failing assertion leaves it running."""
            if proc.poll() is None:
                proc.kill()
            proc.wait(timeout=10)
        self.addCleanup(cleanup)
        deadline = time.monotonic() + 10
        while not ready.exists():
            if proc.poll() is not None or time.monotonic() > deadline:
                self.fail('Isolated process fixture did not start')
            time.sleep(0.05)
        return proc

    def test_fresh_upgrade_reinstall_downgrade_and_running_consent(self):
        """Exercise real setup branches and prove preflight failures preserve old files."""
        for component in ('server', 'desktop'):
            with self.subTest(component=component):
                target = self.fixture['base'] / ('安装 目录-' + component)
                self.addCleanup(self.cleanup_install, target)
                self.assertIsNone(self.installed(component))
                self.assertEqual(self.install(component, '9.0.0', target), 0)
                self.assertEqual(self.installed(component), '9.0.0')
                marker = self.fixture['base'] / (component + '-user-data.txt')
                marker.write_text('keep my library')
                managed = target / ('frontend-dist' if component == 'server' else 'resources/app')
                (managed / 'stale.js').write_text('old resource')
                selected = self.start_fixture(target, component)
                self.assertNotEqual(self.install(component, '9.0.1', target), 0)
                self.assertIsNone(selected.poll(), 'No shutdown consent must preserve running app')
                self.assertEqual(self.installed(component), '9.0.0')
                self.assertTrue((managed / 'stale.js').exists(), 'Preflight must not delete files')
                self.assertEqual(self.install(component, '9.0.1', target, '/CLOSECURATED=1'), 0)
                self.assertEqual(selected.wait(timeout=10), 0)
                self.assertEqual(self.installed(component), '9.0.1')
                self.assertEqual((managed / 'version.txt').read_text(), '9.0.1')
                self.assertFalse((managed / 'stale.js').exists())
                self.assertEqual(self.install(component, '9.0.1', target), 0)
                self.assertNotEqual(self.install(component, '9.0.0', target), 0)
                relocated = self.fixture['base'] / ('relocated-' + component)
                self.assertNotEqual(self.install(component, '9.0.1', relocated), 0)
                self.assertFalse((relocated / 'curated.exe').exists())
                self.assertEqual(marker.read_text(), 'keep my library')
                self.cleanup_install(target)
                self.assertIsNone(self.installed(component))
                self.assertEqual(marker.read_text(), 'keep my library')

    def test_first_install_rejects_nonempty_directory(self):
        """Ensure a fresh install never overwrites a folder owned by someone else."""
        target = self.fixture['base'] / 'unrelated files'
        target.mkdir()
        marker = target / 'keep.txt'
        marker.write_text('unrelated')
        self.assertNotEqual(self.install('desktop', '9.0.0', target), 0)
        self.assertIsNone(self.installed('desktop'))
        self.assertEqual(marker.read_text(), 'unrelated')
