import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from scripts.release.release_lib.native_player import (stage_native_player, validate_native_bundle,
    stage_native_sources, validate_source_archive)


class NativeEnginePayloadTests(unittest.TestCase):
    def fixture(self, root):
        """提供有校验值的最小引擎、许可与来源资料，隔离真实媒体和网络。"""
        def entry(name):
            """写入可用于独立来源 ZIP 校验的 fixture 材料。"""
            file = root / name
            file.parent.mkdir(parents=True, exist_ok=True)
            file.write_bytes(b'{}' if name.endswith('build-inputs.json') else name.encode())
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

    def test_runtime_excludes_sources_and_references_the_verified_companion(self):
        """完整源码独立交付后运行目录仍可验证，且不接受把运行目录当完整准备资料。"""
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            bundle = root / 'bundle'
            bundle.mkdir()
            self.fixture(bundle)
            source = stage_native_sources(bundle, root / 'release', '0.2.5', 'release-20261007-2')
            stage_native_player(root / 'app', bundle, source_asset=source)
            runtime = root / 'app/native-player'
            self.assertFalse((runtime / 'sources/complete.tar.zst').exists())
            self.assertTrue((runtime / 'sources/build-inputs.json').exists())
            manifest = validate_native_bundle(runtime)
            self.assertEqual(manifest['sourceAsset'], source)
            self.assertIn(source['url'], (runtime / 'licenses/SOURCE-DOWNLOAD.txt').read_text())
            with self.assertRaisesRegex(ValueError, 'local corresponding'):
                validate_native_bundle(runtime, prepared=True)
            validate_source_archive(root / 'release' / source['fileName'])
            manifest['sourceAsset']['url'] = 'https://example.test/moving-sources'
            (runtime / 'engine-manifest.json').write_text(json.dumps(manifest))
            with self.assertRaisesRegex(ValueError, 'beside the binary'):
                validate_native_bundle(runtime)

    def test_source_companion_rejects_missing_and_changed_materials(self):
        """独立来源包缺源码或改源码必须失败，不能靠仅提供 URL 放行。"""
        from zipfile import ZipFile
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            bundle = root / 'bundle'
            bundle.mkdir()
            self.fixture(bundle)
            source = stage_native_sources(bundle, root / 'release', '0.2.5', 'release-20261007-2')
            with ZipFile(root / 'release' / source['fileName']) as original:
                for missing in (True, False):
                    bad = root / ('missing.zip' if missing else 'changed.zip')
                    with ZipFile(bad, 'w') as archive:
                        for name in original.namelist():
                            if name == 'sources/complete.tar.zst' and missing:
                                continue
                            archive.writestr(name, b'changed' if name == 'sources/complete.tar.zst' else original.read(name))
                    with self.assertRaises(ValueError):
                        validate_source_archive(bad)

    def test_production_windows_requires_a_prepared_engine(self):
        with patch.dict('os.environ', {}, clear=True), self.assertRaisesRegex(ValueError, 'requires a prepared'):
            stage_native_player(Path('unused'), required=True)


if __name__ == '__main__':
    unittest.main()
