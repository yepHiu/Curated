import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from scripts.release.release_lib.macos_native import validate_bundle, stage, safe_file, system_library
from scripts.release.release_lib.native_materials import digest, lock_digest


class MacNativeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.bundle = self.root / 'bundle'
        self.bundle.mkdir()
        self.lock = self.root / 'lock.json'
        def material(name, value):
            file = self.bundle / name
            file.parent.mkdir(parents=True, exist_ok=True)
            file.write_text(value)
            return {'file': name, 'sha256': digest(file)}
        source = material('sources/source', 'source bytes')
        source_path = self.bundle / 'sources' / source['sha256']
        (self.bundle / source['file']).rename(source_path)
        source['file'] = 'sources/' + source['sha256']
        lock = {'runtime': [{'file': 'libmpv.dylib'}], 'packages': {'mpv': {'sources': [{'sha256': source['sha256']}], 'recipeSha256': hashlib.sha256(b'recipe').hexdigest()}}, 'minimumMacOS': '15.0'}
        self.lock.write_text(json.dumps(lock))
        self.manifest = {'schema': 1, 'engine': 'libmpv', 'arch': 'arm64', 'platform': 'macos', 'minimumMacOS': '15.0',
            'lockSha256': lock_digest(self.lock), 'runtime': [material('libmpv.dylib', 'binary')],
            'buildInputs': material('build-inputs.json', json.dumps({'lock': lock})),
            'materials': [source, material('licenses/mpv/Copyright', 'copyright'),
                          material('metadata/mpv/recipe.rb', 'recipe'), material('metadata/mpv/sbom.spdx.json', '{}')]}
        self.save()

    def save(self):
        (self.bundle / 'engine-manifest.json').write_text(json.dumps(self.manifest))

    def test_complete_pinned_materials_pass(self):
        validate_bundle(self.bundle, self.lock, inspect_macho=False)

    def test_missing_library_or_corrupt_sources_fail(self):
        (self.bundle / self.manifest['materials'][0]['file']).write_text('tampered')
        with self.assertRaisesRegex(ValueError, 'checksum'):
            validate_bundle(self.bundle, self.lock, inspect_macho=False)

    def test_absent_recipe_and_wrong_lock_fail(self):
        self.manifest['materials'] = self.manifest['materials'][:2]
        self.save()
        with self.assertRaisesRegex(ValueError, 'incomplete'):
            validate_bundle(self.bundle, self.lock, inspect_macho=False)
        self.lock.write_text('{}')
        with self.assertRaisesRegex(ValueError, 'release lock'):
            validate_bundle(self.bundle, self.lock, inspect_macho=False)

    def test_no_engine_cannot_produce_mac_package(self):
        with patch.dict('os.environ', {}, clear=True), self.assertRaisesRegex(ValueError, 'requires an absolute'):
            stage(self.root / 'app', self.lock)

    def test_external_dependencies_rejected(self):
        with patch('scripts.release.release_lib.macos_native.run', return_value='arm64'), \
             patch('scripts.release.release_lib.macos_native.dependencies', return_value=['/opt/homebrew/lib/libfoo.dylib']), \
             self.assertRaisesRegex(ValueError, 'External Mac native dependency'):
            validate_bundle(self.bundle, self.lock)

    def test_path_escape_and_system_prefix_confusion(self):
        for name in ('../outside', '/outside', 'a\\b'):
            with self.assertRaises(ValueError): safe_file(self.bundle, name)
        self.assertTrue(system_library('/usr/lib/libSystem.B.dylib'))
        self.assertFalse(system_library('/usr/library/third-party.dylib'))
        self.assertFalse(system_library('/usr/lib/../../tmp/third-party.dylib'))

    def test_library_set_and_recipe_are_pinned(self):
        self.manifest['runtime'][0]['file'] = 'other.dylib'
        self.save()
        with self.assertRaisesRegex(ValueError, 'incomplete'):
            validate_bundle(self.bundle, self.lock, inspect_macho=False)
        self.manifest['runtime'][0]['file'] = 'libmpv.dylib'
        recipe = self.bundle / 'metadata/mpv/recipe.rb'
        recipe.write_text('different recipe')
        self.manifest['materials'][2]['sha256'] = digest(recipe)
        self.save()
        with self.assertRaisesRegex(ValueError, 'recipe differs'):
            validate_bundle(self.bundle, self.lock, inspect_macho=False)

    def test_rpaths_and_newer_system_requirement_fail(self):
        for load_commands, error in [('cmd LC_RPATH\ncmdsize 48\npath /usr/local/lib (offset 12)', 'rpath'),
                                     ('cmd LC_BUILD_VERSION\nminos 16.0', 'newer macOS')]:
            with self.subTest(error=error), \
                 patch('scripts.release.release_lib.macos_native.run', side_effect=['arm64', load_commands]), \
                 patch('scripts.release.release_lib.macos_native.dependencies', return_value=['@loader_path/libmpv.dylib']), \
                 self.assertRaisesRegex(ValueError, error):
                validate_bundle(self.bundle, self.lock)
