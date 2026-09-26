"""Validate that the production Full build embeds the migration executable."""
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from scripts.release.release_lib import windows_components as windows

ROOT = Path(__file__).resolve().parents[3]


class MigrationPackagingTests(unittest.TestCase):
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
