import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from urllib.error import HTTPError

from scripts.release import cd_release as cd


class CDReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.output = self.root / '.workspace/cd-release'
        self.tag = 'v1.5.8'
        self.version = '1.5.8'
        self.notes = self.root / 'docs/release-notes/2026-09-27-release-1.5.8-notes.md'
        self.notes.parent.mkdir(parents=True)
        self.notes.write_text('# Internal metadata\n\n## GitHub Release Body\n\nRelease summary.\n\n### Downloads\n\nPackages.\n', encoding='utf-8')
        self.version_file = self.root / 'scripts/release/version.json'
        self.version_file.parent.mkdir(parents=True)
        self.version_file.write_text(json.dumps({'schema': 1, 'current': {'major': 1, 'minor': 5, 'patch': 8}}))
        for args in [('init', '-q'), ('config', 'user.email', 'cd@example.test'), ('config', 'user.name', 'CD test'),
                     ('add', '.'), ('commit', '-qm', 'Release source'), ('tag', '-a', self.tag, '-m', self.tag)]:
            cd.git(self.root, *args)
        self.commit = cd.git(self.root, 'rev-parse', 'HEAD')
        self.metadata = {'tag': self.tag, 'version': self.version, 'commit': self.commit}
        self.manifest_path = self.root / 'release/manifest/release.json'
        self.manifest_path.parent.mkdir(parents=True)
        self.manifest = {'version': self.version, 'channel': 'release', 'artifacts': []}
        for kind, name in zip(('portable', 'installer'), cd.artifact_names(self.version)):
            path = self.root / 'release' / kind / name
            path.parent.mkdir(parents=True)
            path.write_bytes(f'Binary fixture for {kind}'.encode())
            self.manifest['artifacts'].append({'type': kind, 'fileName': name, 'path': str(path), 'sha256': cd.sha256(path)})
        self.write_manifest()

    def write_manifest(self):
        self.manifest_path.write_text(json.dumps(self.manifest), encoding='utf-8')

    def test_only_canonical_release_tags_are_allowed(self):
        self.assertEqual(cd.version_from_tag(self.tag), self.version)
        for tag in ('1.5.8', 'v01.5.8', 'v1.5.8-beta.1', 'v1.5', '--help', 'v1.5.8\n', '../v1.5.8', 'v1.5.8; echo bad'):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                cd.version_from_tag(tag)

    def test_prepare_resolves_annotated_tag_and_does_not_allocate_version(self):
        before = self.version_file.read_bytes()
        self.assertEqual(cd.prepare(self.root, self.tag, self.output), self.metadata)
        self.assertEqual(self.version_file.read_bytes(), before)
        self.assertEqual((self.output / 'release-body.md').read_text(), 'Release summary.\n\n### Downloads\n\nPackages.\n')

    def test_prepare_rejects_wrong_checkout(self):
        cd.git(self.root, 'commit', '--allow-empty', '-qm', 'Later commit')
        with self.assertRaisesRegex(ValueError, 'Checkout'):
            cd.prepare(self.root, self.tag, self.output)

    def test_prepare_rejects_unsynchronized_version(self):
        self.version_file.write_text('{"schema":1,"current":{"major":1,"minor":5,"patch":7}}')
        with self.assertRaisesRegex(ValueError, 'must agree'):
            cd.prepare(self.root, self.tag, self.output)

    def test_notes_are_required_and_unique(self):
        extra = self.notes.with_name('2026-09-28-release-1.5.8-notes.md')
        extra.write_text(self.notes.read_text())
        with self.assertRaisesRegex(ValueError, 'exactly one release notes'):
            cd.release_body(self.root, self.version)
        extra.unlink()
        self.notes.unlink()
        with self.assertRaisesRegex(ValueError, 'exactly one release notes'):
            cd.release_body(self.root, self.version)

    def test_notes_reject_missing_duplicate_and_empty_bodies(self):
        for text in ('# No body', '## GitHub Release Body\n\n',
                     '## GitHub Release Body\n<!-- placeholder -->',
                     '## GitHub Release Body\nBody\n## GitHub Release Body\nMore'):
            with self.subTest(text=text), self.assertRaises(ValueError):
                self.notes.write_text(text)
                cd.release_body(self.root, self.version)

    def test_stage_verifies_both_packages_and_removes_runner_paths(self):
        original = self.manifest_path.read_bytes()
        cd.stage(self.root, self.tag, self.output)
        cd.verify_staged(self.output, self.metadata)
        manifest = json.loads((self.output / 'assets/release.json').read_text())
        self.assertEqual(manifest['sourceCommit'], self.commit)
        for entry in manifest['artifacts']:
            self.assertEqual(entry['path'], entry['fileName'])
        self.assertEqual(self.manifest_path.read_bytes(), original)

    def test_partial_build_cannot_stage(self):
        self.manifest['artifacts'].pop()
        self.write_manifest()
        with self.assertRaisesRegex(ValueError, 'Both installer and portable'):
            cd.stage(self.root, self.tag, self.output)

    def test_missing_empty_or_modified_package_cannot_stage(self):
        entry = self.manifest['artifacts'][1]
        path = Path(entry['path'])
        for content in (b'', b'tampered', None):
            if content is None:
                path.unlink()
            else:
                path.write_bytes(content)
            with self.subTest(content=content), self.assertRaises(ValueError):
                cd.stage(self.root, self.tag, self.output)

    def test_manifest_identity_and_names_must_match(self):
        for field, value in [('version', '1.5.7'), ('channel', 'dev')]:
            original = self.manifest[field]
            self.manifest[field] = value
            self.write_manifest()
            with self.assertRaises(ValueError):
                cd.stage(self.root, self.tag, self.output)
            self.manifest[field] = original
        self.manifest['artifacts'][0]['fileName'] = '../unrelated.zip'
        self.write_manifest()
        with self.assertRaises(ValueError):
            cd.stage(self.root, self.tag, self.output)

    def test_transport_corruption_is_rejected(self):
        cd.stage(self.root, self.tag, self.output)
        (self.output / 'assets/release.json').write_text('{}')
        with self.assertRaisesRegex(ValueError, 'checksum'):
            cd.verify_staged(self.output, self.metadata)

    def test_remote_tag_changes_and_existing_public_releases_are_rejected(self):
        ref = {'object': {'type': 'commit', 'sha': self.commit}}
        with patch.object(cd, 'api', side_effect=[ref, {'draft': False}]), self.assertRaisesRegex(ValueError, 'already published'):
            cd.check_release(self.metadata)
        with patch.object(cd, 'api', return_value={'object': {'type': 'commit', 'sha': 'wrong'}}), self.assertRaisesRegex(ValueError, 'Remote tag'):
            cd.check_release(self.metadata)
        with patch.object(cd, 'api', side_effect=[ref, {'draft': True, 'target_commitish': 'master'}]), self.assertRaisesRegex(ValueError, 'unpinned'):
            cd.check_release(self.metadata)

    def test_annotated_remote_tag_and_same_commit_draft_can_resume(self):
        draft = {'draft': True, 'target_commitish': 'master', 'body': cd.source_marker(self.metadata)}
        with patch.object(cd, 'api', side_effect=[{'object': {'type': 'tag', 'sha': 'tag-object'}},
                                                {'object': {'type': 'commit', 'sha': self.commit}}, draft]):
            self.assertEqual(cd.check_release(self.metadata), draft)

    def upload_response(self):
        return {'id': 123, 'tag_name': self.tag, 'draft': True, 'target_commitish': self.commit, 'body': cd.source_marker(self.metadata), 'html_url': 'https://example.test/release',
                'assets': [{'name': p.name, 'size': p.stat().st_size, 'digest': f'sha256:{cd.sha256(p)}'}
                           for p in (self.output / 'assets').iterdir()]}

    def test_publish_only_goes_public_after_upload_digest_verification(self):
        cd.stage(self.root, self.tag, self.output)
        release = self.upload_response()
        ref = {'object': {'type': 'commit', 'sha': self.commit}}
        with patch.object(cd, 'source_commit', return_value=self.commit), \
             patch.object(cd, 'api', side_effect=[ref, None, [], release, release, ref, None, [release], release]) as api, \
             patch.object(cd.subprocess, 'run') as run, \
             patch.dict(os.environ, {'GITHUB_REPOSITORY': 'owner/repo'}):
            cd.publish(self.root, self.tag, self.output, 'publish')
        self.assertIn('--clobber', run.call_args.args[0])
        self.assertEqual(api.call_args.args[1], {'draft': False, 'make_latest': 'true'})

    def test_manual_draft_does_not_publish(self):
        cd.stage(self.root, self.tag, self.output)
        release = self.upload_response()
        ref = {'object': {'type': 'commit', 'sha': self.commit}}
        with patch.object(cd, 'source_commit', return_value=self.commit), \
             patch.object(cd, 'api', side_effect=[ref, None, [release], release, release]) as api, \
             patch.object(cd.subprocess, 'run'), \
             patch.dict(os.environ, {'GITHUB_REPOSITORY': 'owner/repo'}):
            cd.publish(self.root, self.tag, self.output, 'draft')
        writes = [call.args[1] for call in api.call_args_list if len(call.args) > 1]
        self.assertTrue(all(payload['draft'] for payload in writes))

    def test_bad_uploaded_digest_leaves_release_as_draft(self):
        cd.stage(self.root, self.tag, self.output)
        release = self.upload_response()
        release['assets'][0]['digest'] = 'sha256:wrong'
        ref = {'object': {'type': 'commit', 'sha': self.commit}}
        with patch.object(cd, 'source_commit', return_value=self.commit), \
             patch.object(cd, 'api', side_effect=[ref, None, [], release, release]) as api, \
             patch.object(cd.subprocess, 'run'), \
             patch.dict(os.environ, {'GITHUB_REPOSITORY': 'owner/repo'}), \
             self.assertRaisesRegex(ValueError, 'upload verification'):
            cd.publish(self.root, self.tag, self.output, 'publish')
        self.assertFalse(any(len(call.args) > 1 and call.args[1].get('draft') is False for call in api.call_args_list))

    def test_draft_lookup_paginates_when_tag_endpoint_returns_404(self):
        draft = {'id': 123, 'draft': True, 'tag_name': self.tag}
        with patch.object(cd, 'api', side_effect=[None, [{'tag_name': 'other'}] * 100, [draft]]) as api:
            self.assertEqual(cd.find_release(self.tag), draft)
        self.assertEqual(api.call_args.args[0], 'releases?per_page=100&page=2')

    def test_draft_listing_permission_failure_is_not_treated_as_no_release(self):
        error = HTTPError('https://example.test', 403, 'forbidden', {}, None)
        self.addCleanup(error.close)
        with patch.object(cd, 'api', side_effect=[None, error]), self.assertRaises(HTTPError):
            cd.find_release(self.tag)

    def test_api_only_treats_release_lookup_404_as_absent(self):
        with patch.dict(os.environ, {'GITHUB_REPOSITORY': 'owner/repo', 'GH_TOKEN': 'test-token'}):
            for status in (403, 404, 500):
                error = HTTPError('https://example.test', status, 'failure', {}, None)
                self.addCleanup(error.close)
                with patch.object(cd.urllib.request, 'urlopen', side_effect=error):
                    if status == 404:
                        self.assertIsNone(cd.api('releases/tags/v1.5.8'))
                    else:
                        with self.assertRaises(HTTPError):
                            cd.api('releases/tags/v1.5.8')


if __name__ == '__main__':
    unittest.main()
