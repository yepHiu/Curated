"""Validate that the production Full build embeds the migration executable."""
import tempfile
import subprocess
import unittest
from pathlib import Path
from unittest.mock import patch

from scripts.release.release_lib import windows_components as windows

ROOT = Path(__file__).resolve().parents[3]


class MigrationPackagingTests(unittest.TestCase):
    @unittest.skipUnless(windows._find_iscc(), 'Inno Setup is not installed')
    def test_real_compiler_accepts_all_localized_installers(self):
        with tempfile.TemporaryDirectory(prefix='curated-installer-compile-') as temporary:
            work = Path(temporary)
            payload = work / 'payload'
            payload.mkdir()
            (payload / 'curated.exe').write_bytes(b'compile-only fixture, never executed')
            (payload / 'curated.ico').write_bytes((ROOT / 'icon/curated-desktop.ico').read_bytes())
            values = {'APP_ID': 'Curated.CompileFixture', 'SOURCE': str(payload), 'EXE': 'curated.exe',
                      'PARAMS': '', 'MANAGED_DELETE': 'frontend-dist', 'LEGACY_CHECK': 'False',
                      'RUN_FLAGS': 'nowait postinstall skipifsilent', 'MIGRATION_HELPER': str(payload / 'curated.exe'),
                      'SERVER_INSTALLER': str(payload / 'curated.exe'), 'DESKTOP_INSTALLER': str(payload / 'curated.exe'),
                      'SERVER_VERSION': '9.0.0', 'DESKTOP_VERSION': '9.0.0'}
            def compile_quietly(command, cwd):
                result = subprocess.run(command[:1] + ['/Q'] + command[1:], cwd=cwd, capture_output=True)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            with patch.object(windows, '_run', side_effect=compile_quietly):
                for component in ('server', 'desktop', 'full'):
                    with self.subTest(component=component):
                        self.assertTrue(windows.compile_installer(ROOT, work, work, component, '9.0.0', values).is_file())
            legacy = (ROOT / 'scripts/release/windows/Curated.iss.tpl').read_text(encoding='utf-8')
            for key, value in {'WINDOWS_SUPPORT': str(ROOT / 'scripts/release/windows'), 'APP_VERSION': '9.0.0',
                               'APP_DIR': str(payload), 'OUTPUT_DIR': str(work), 'SETUP_BASENAME': 'legacy-fixture'}.items():
                legacy = legacy.replace(f'__{key}__', value)
            source = work / 'legacy.iss'
            source.write_text(legacy, encoding='utf-8-sig')
            compile_quietly([str(windows._find_iscc()), str(source)], ROOT)

    def test_full_compiles_with_migration_helper_and_actual_component_versions(self):
        with tempfile.TemporaryDirectory() as temporary:
            work = Path(temporary)
            output = work / 'output'
            output.mkdir()
            helper = work / 'curated-migrate.exe'
            helper.write_bytes(b'PE placeholder')
            seen = []

            def compile_source(command, cwd):
                source = Path(command[1]).read_text(encoding='utf-8-sig')
                seen.append(source)
                # Simulate only the compiler boundary; verify the actual template input.
                self.assertIn(str(helper), source)
                self.assertIn("InstallComponent('Server', '1.7.2'", source)
                self.assertIn("InstallComponent('Desktop', '0.2.0'", source)
                self.assertLess(source.index("if not RunMigration('prepare')"),
                                source.index("ServerOK := InstallComponent"))
                self.assertLess(source.index("DesktopOK := InstallComponent"),
                                source.index("if not RunMigration('complete')"))
                self.assertNotIn('__MIGRATION_HELPER__', source)
                (output / 'Curated-Full-Setup-1.7.2-windows-x64.exe').write_bytes(b'installer')

            values = {'MIGRATION_HELPER': str(helper), 'SERVER_INSTALLER': str(work / 'server.exe'),
                      'DESKTOP_INSTALLER': str(work / 'desktop.exe'),
                      'SERVER_VERSION': '1.7.2', 'DESKTOP_VERSION': '0.2.0'}
            with patch.object(windows, '_find_iscc', return_value=Path('ISCC.exe')), \
                 patch.object(windows, '_run', side_effect=compile_source):
                result = windows.compile_installer(ROOT, work, output, 'full', '1.7.2', values)
            self.assertTrue(result.is_file())
            self.assertEqual(len(seen), 1)
            del values['MIGRATION_HELPER']
            with patch.object(windows, '_find_iscc', return_value=Path('ISCC.exe')), \
                 patch.object(windows, '_run') as run:
                with self.assertRaisesRegex(ValueError, 'Unresolved installer'):
                    windows.compile_installer(ROOT, work, output, 'full', '1.7.2', values)
                run.assert_not_called()
