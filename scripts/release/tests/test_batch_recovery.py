import base64
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from scripts.release import recover_release, component_cd as cd
from scripts.release.release_lib import sync_notes


class BatchRecoveryTests(unittest.TestCase):
    def test_bridge_tag_recovery_requires_mac_gate_when_desktop_was_selected(self):
        tag = 'server-v1.7.7'
        batch = {'schema': 1, 'tag': tag, 'modules': {'server': {'changed': True}, 'desktop': {'changed': True}}}
        names = ['Build Windows x64 packages', 'Validate release source and notes',
                 'Check the exact release commit / Frontend, Electron, and release scripts',
                 'Check the exact release commit / Go test and vet', 'Check the exact release commit / Runtime browser e2e']
        jobs = [{'name': n, 'conclusion': 'success'} for n in names]
        def gh(path, **_):
            if path.endswith('/jobs?per_page=100'): return {'jobs': jobs}
            if '/git/trees/' in path: return {'tree': [{'path': f'scripts/release/batches/{tag}.json', 'sha': 'blob'}]}
            if '/git/blobs/' in path: return {'content': base64.b64encode(json.dumps(batch).encode()).decode()}
            if '/releases?' in path: return [{'tag_name': tag, 'draft': False}]
            return {'path': '.github/workflows/cd-release.yml', 'conclusion': 'failure', 'event': 'push',
                    'head_branch': tag, 'head_sha': 'a' * 40, 'run_attempt': 2}
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / 'output'
            with patch.object(recover_release, 'gh', side_effect=gh), patch.dict(os.environ, {
                'REQUEST_RUN': '123', 'REQUEST_MODE': 'publish', 'GITHUB_REPOSITORY': 'yepHiu/Curated', 'GITHUB_OUTPUT': str(output)}):
                with self.assertRaisesRegex(ValueError, 'Mac Desktop'):
                    recover_release.main()
                self.assertFalse(output.exists())
                jobs.append({'name': 'Build Mac Desktop (Apple Silicon)', 'conclusion': 'success'})
                recover_release.main()
                text = output.read_text()
                self.assertIn('command=channels\n', text)
                self.assertIn('component=both\n', text)
                self.assertIn(f'macos_artifact=macos-desktop-{tag}-2\n', text)

    def test_notes_sync_preserves_batch_title_and_unchanged_component_links(self):
        batch = {'id': '20261001-2', 'tag': 'release-20261001-2', 'modules': {
            'server': {'changed': True, 'before': '1.7.7', 'after': '1.7.8'},
            'desktop': {'changed': False, 'before': '0.2.3', 'after': '0.2.3', 'previousArtifacts': [
                {'url': 'https://github.com/yepHiu/Curated/releases/download/release-20261001/desktop.dmg'}]},
        }}
        current_url = 'https://github.com/yepHiu/Curated/releases/download/release-20261001-2/server.exe'
        body = ('| Server | Updated | 1.7.7 | 1.7.8 |\n| Desktop | Unchanged | 0.2.3 | 0.2.3 |\n'
                + current_url + '\n' + batch['modules']['desktop']['previousArtifacts'][0]['url'])
        release = {'id': 1, 'tag_name': batch['tag'], 'target_commitish': 'master', 'draft': False, 'prerelease': False,
                   'published_at': '2026-10-01', 'name': 'old', 'body': cd.legacy.source_marker({'commit': 'a' * 40}),
                   'assets': [{'id': 1, 'name': 'server.exe', 'size': 10, 'browser_download_url': current_url}], 'html_url': 'fixture'}
        def api(path, payload=None, method=None):
            if payload: release.update(payload)
            return dict(release)
        def git(root, *args):
            return json.dumps(batch) if args[0] == 'show' else 'a' * 40
        with patch.dict(os.environ, {'RELEASE_TAG': batch['tag']}), patch.object(sync_notes, 'load_batch', return_value=batch), \
             patch.object(cd.legacy, 'api', side_effect=api), patch.object(cd.legacy, 'git', side_effect=git), \
             patch.object(cd, 'body', return_value=body):
            sync_notes.main()
        self.assertEqual(release['name'], 'Curated 20261001-2')
        self.assertEqual(release['body'], body)

    def test_duplicate_date_sequence_is_rejected_before_publication(self):
        meta = {'batch': {'id': '20261001'}, 'tag': 'release-20261001'}
        with patch.object(cd.legacy, 'api', return_value=[{
            'tag_name': 'server-v1.7.7', 'body': '<!-- curated-release-batch:20261001 -->'}]), \
             self.assertRaisesRegex(ValueError, 'already reserved'):
            cd.check_batch_identity(meta)
