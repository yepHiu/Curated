import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from scripts.release.release_lib.native_player import stage_native_player, validate_native_bundle


class NativeEnginePayloadTests(unittest.TestCase):
    def fixture(self, root):
        def entry(name):
            file = root / name
            file.parent.mkdir(parents=True, exist_ok=True)
            file.write_bytes(name.encode())
            return {'file': name, 'sha256': hashlib.sha256(file.read_bytes()).hexdigest()}
        value = {'schema': 1, 'engine': 'mpv', 'platform': 'windows', 'arch': 'x64',
                 'runtime': [entry('mpv.exe')],
                 'licenses': [entry('licenses/COPYING')],
                 'correspondingSource': entry('sources/complete.tar.zst'),
                 'buildInputs': entry('sources/build-inputs.json'),
                 'components': [{'name': name, 'revision': 'exact-commit', 'license': 'GPL', 'sourcePath': name}
                                for name in ('mpv', 'ffmpeg')]}
        (root / 'engine-manifest.json').write_text(json.dumps(value))
        return value

    def test_stages_only_prepared_manifest_files(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / 'bundle'
            source.mkdir()
            self.fixture(source)
            (source / 'private.txt').write_text('not distributable')
            self.assertTrue(stage_native_player(root / 'app', source))
            self.assertFalse((root / 'app/native-player/private.txt').exists())
            validate_native_bundle(root / 'app/native-player')

    def test_refuses_missing_sources_changed_binaries_and_escaping_paths(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            value = self.fixture(root)
            (root / 'mpv.exe').write_text('changed')
            with self.assertRaisesRegex(ValueError, 'checksum'):
                validate_native_bundle(root)
            self.fixture(root)
            (root / 'sources/complete.tar.zst').unlink()
            with self.assertRaisesRegex(ValueError, 'Missing'):
                validate_native_bundle(root)
            value['correspondingSource'] = {'file': '../outside', 'sha256': 'a' * 64}
            (root / 'engine-manifest.json').write_text(json.dumps(value))
            with self.assertRaisesRegex(ValueError, 'escapes'):
                validate_native_bundle(root)

    def test_optional_stager_returns_false_without_a_prepared_engine(self):
        with patch.dict('os.environ', {}, clear=True):
            self.assertFalse(stage_native_player(Path('unused')))

    def test_production_windows_requires_a_prepared_engine(self):
        with patch.dict('os.environ', {}, clear=True), self.assertRaisesRegex(ValueError, 'requires a prepared'):
            stage_native_player(Path('unused'), required=True)


if __name__ == '__main__':
    unittest.main()
