import hashlib
import importlib.util
import io
import json
import subprocess
from pathlib import Path
import tarfile
import tempfile
import unittest
from zipfile import ZipFile
from scripts.release.release_lib.native_materials import download, stage_package, write_source_archive, read_source_index

ROOT = Path(__file__).resolve().parents[3]
RESTORE_SCRIPT = ROOT / 'scripts/release/native-player/restore-sources.py'


def package(file, entries):
    with tarfile.open(file, 'w:gz') as archive:
        for name, data in entries.items():
            entry = tarfile.TarInfo(name)
            entry.size = len(data)
            entry.mode = 0o644
            archive.addfile(entry, io.BytesIO(data))


class NativeMaterialTests(unittest.TestCase):
    def test_shallow_git_sources_preserve_commit_tree_and_build_version(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            checkout = root / 'checkout'
            checkout.mkdir()
            def git(where, *args):
                return subprocess.check_output(['git', '-c', 'user.name=Source fixture', '-c', 'user.email=fixture@example.invalid', '-C', str(where), *args], text=True).strip()
            git(checkout, 'init', '-q', '-b', 'master')
            for value in ('one', 'two', 'three'):
                (checkout / 'source.c').write_text(value)
                git(checkout, 'add', 'source.c')
                git(checkout, 'commit', '-q', '-m', value)
                if value == 'one':
                    git(checkout, 'tag', 'v1.0')
            revision = git(checkout, 'rev-parse', 'HEAD')
            tree = git(checkout, 'rev-parse', 'HEAD^{tree}')
            describe = git(checkout, 'describe', '--long', '--tags', '--abbrev=7')
            snapshot = root / 'upstream/fixture/cache'
            snapshot.parent.mkdir(parents=True)
            subprocess.run(['git', 'clone', '--quiet', '--bare', str(checkout), str(snapshot)], check=True)
            (snapshot.parent / 'PKGBUILD').write_text('original recipe')
            source = root / 'source.tar.gz'
            with tarfile.open(source, 'w:gz') as archive:
                archive.add(snapshot.parent, arcname='fixture')
            value = {'id': 'fixture-1.0', 'base': 'fixture', 'file': source.name,
                     'sha256': hashlib.sha256(source.read_bytes()).hexdigest(),
                     'gitSnapshot': {'path': 'fixture/cache', 'revision': revision, 'describe': describe, 'abbrev': 7, 'depth': 3}}
            complete = root / 'complete.zip'
            write_source_archive(complete, [value], [], root, RESTORE_SCRIPT)
            spec = importlib.util.spec_from_file_location('restore_git_sources', RESTORE_SCRIPT)
            module = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(module)
            module.restore(complete, root / 'restored')
            restored = root / 'restored/snapshots/fixture-1.0/fixture/cache'
            self.assertEqual(git(restored, 'rev-parse', revision + '^{tree}'), tree)
            self.assertEqual(git(restored, 'describe', '--long', '--tags', revision), describe)
            self.assertEqual((restored.parent / 'PKGBUILD').read_text(), 'original recipe')
            self.assertTrue((restored / 'shallow').exists())
            self.assertNotIn('curated-source-git-', (restored / 'config').read_text())

    def test_zstandard_source_archive_retains_catalogue_and_restores_offline(self):
        try:
            from compression import zstd
        except ImportError:
            try:
                import zstandard
            except ImportError:
                self.skipTest('Zstandard installed in Windows production job')
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            package(root / 'source.tar.gz', {'project/PKGBUILD': b'build the exact source', 'project/source.c': b'int main(void) { return 0; }'})
            file = root / 'complete.tar.zst'
            write_source_archive(file, [{'id': 'source', 'file': 'source.tar.gz'}], [], root, RESTORE_SCRIPT)
            index, crates = read_source_index(file)
            self.assertIn('source', index)
            self.assertEqual(crates, [])
            spec = importlib.util.spec_from_file_location('restore_zstandard_sources', RESTORE_SCRIPT)
            module = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(module)
            module.restore(file, root / 'restored')
            self.assertEqual((root / 'restored/snapshots/source/project/source.c').read_bytes(), b'int main(void) { return 0; }')

    def test_runtime_is_selected_by_archive_path_and_verified_against_recipe(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source, output = root / 'package.tar.gz', root / 'output'
            output.mkdir()
            data = b'pinned executable'
            package(source, {'ucrt64/bin/mpv.exe': data, 'ucrt64/bin/ffmpeg.exe': b'not selected',
                             'ucrt64/share/licenses/mpv/COPYING': b'license',
                             '.BUILDINFO': b'pkgbuild_sha256sum = recipe\n', '.PKGINFO': b'pkgver = 1\n'})
            value = {'runtime': [{'archivePath': 'ucrt64/bin/mpv.exe', 'file': 'mpv.exe',
                                 'sha256': hashlib.sha256(data).hexdigest()}], 'recipeSha256': 'recipe'}
            records, licenses = stage_package(value, source, output)
            self.assertIn('.BUILDINFO', records)
            self.assertEqual(len(licenses), 1)
            self.assertFalse((output / 'ffmpeg.exe').exists())
            value['recipeSha256'] = 'different'
            with self.assertRaisesRegex(ValueError, 'recipe differs'):
                stage_package(value, source, output)
            value['runtime'][0]['sha256'] = '0' * 64
            with self.assertRaisesRegex(ValueError, 'differs from frozen'):
                stage_package(value, source, output)

    def test_source_restoration_preserves_files_and_deduplicates_shared_inputs(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for name in ('one', 'two'):
                package(root / (name + '.tar.gz'), {'project/PKGBUILD': name.encode(),
                                                    'project/upstream.tar.gz': b'same exact source archive'})
            source = root / 'complete.zip'
            write_source_archive(source, [{'id': name, 'file': name + '.tar.gz'} for name in ('one', 'two')], [], root, RESTORE_SCRIPT)
            repeat = root / 'repeat.zip'
            write_source_archive(repeat, [{'id': name, 'file': name + '.tar.gz'} for name in ('one', 'two')], [], root, RESTORE_SCRIPT)
            self.assertEqual(source.read_bytes(), repeat.read_bytes())
            with ZipFile(source) as archive:
                self.assertEqual(len([name for name in archive.namelist() if name.startswith('blobs/')]), 3)
            spec = importlib.util.spec_from_file_location('restore_native_sources', RESTORE_SCRIPT)
            module = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(module)
            module.restore(source, root / 'restored')
            for name in ('one', 'two'):
                self.assertEqual((root / 'restored/snapshots' / name / 'project/upstream.tar.gz').read_bytes(), b'same exact source archive')

    def test_rejects_source_traversal_and_corrupt_download_cache(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            package(root / 'unsafe.tar.gz', {'../../outside': b'unsafe'})
            with self.assertRaisesRegex(ValueError, 'escapes'):
                write_source_archive(root / 'complete.zip', [{'id': 'unsafe', 'file': 'unsafe.tar.gz'}], [], root, RESTORE_SCRIPT)
            (root / 'input').write_bytes(b'changed')
            with self.assertRaisesRegex(ValueError, 'HTTPS'):
                download({'file': 'input', 'url': 'http://untrusted', 'sha256': '0' * 64}, root)

    def test_cargo_sources_restore_with_offline_checksum_metadata(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            file = root / 'crate-1.0.0.crate'
            package(file, {'crate-1.0.0/Cargo.toml': b'[package]\nname="crate"\nversion="1.0.0"\n',
                           'crate-1.0.0/src/lib.rs': b'pub fn answer() -> u32 { 42 }'})
            value = {'file': file.name, 'name': 'crate', 'version': '1.0.0', 'sha256': hashlib.sha256(file.read_bytes()).hexdigest()}
            source = root / 'complete.zip'
            write_source_archive(source, [], [value], root, RESTORE_SCRIPT)
            spec = importlib.util.spec_from_file_location('restore_native_crates', RESTORE_SCRIPT)
            module = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(module)
            module.restore(source, root / 'restored')
            metadata = json.loads((root / 'restored/vendor/crate-1.0.0/.cargo-checksum.json').read_text())
            self.assertEqual(metadata['package'], value['sha256'])
            self.assertIn('src/lib.rs', metadata['files'])
            self.assertIn('offline = true', (root / 'restored/cargo-home/config.toml').read_text())
