import unittest
from unittest.mock import Mock

from scripts.release.release_lib.latest_release import select_latest, reconcile_latest


def release(tag, identifier=1, **overrides):
    full = tag.startswith('full-v')
    version = tag.removeprefix('full-v') if full else tag.removeprefix('v')
    name = f'Curated-Full-Setup-{version}-windows-x64.exe' if full else f'Curated-Setup-{version}.exe'
    return {'id': identifier, 'tag_name': tag, 'draft': False, 'prerelease': False,
            'assets': [{'name': name, 'size': 100}], **overrides}


class LatestReleaseTests(unittest.TestCase):
    def test_full_wins_over_legacy_components_and_publication_order(self):
        wanted = release('full-v1.10.0', 10)
        releases = [release('server-v9.0.0'), release('desktop-v20.0.0'),
                    release('v99.0.0'), release('full-v1.9.0'), wanted,
                    release('full-v2.0.0', draft=True), release('full-v3.0.0', prerelease=True)]
        self.assertEqual(select_latest(releases), wanted)

    def test_legacy_fallback_and_no_candidate(self):
        self.assertEqual(select_latest([release('v1.5.8'), release('v1.5.7')])['tag_name'], 'v1.5.8')
        with self.assertRaisesRegex(ValueError, 'No public'):
            select_latest([release('server-v1.7.3'), release('full-v1.7.3', draft=True)])

    def test_incomplete_highest_release_is_not_silently_skipped(self):
        for assets in ([], [{'name': 'Curated-Full-Setup-1.7.3-windows-x64.exe', 'size': 0}],
                       [{'name': 'Curated-Desktop-Setup-0.2.1-windows-x64.exe', 'size': 100}]):
            with self.subTest(assets=assets), self.assertRaisesRegex(ValueError, 'missing'):
                select_latest([release('full-v1.7.3', assets=assets), release('v1.5.8')])
        with self.assertRaisesRegex(ValueError, 'Ambiguous'):
            select_latest([release('full-v1.7.3'), release('full-v1.7.3', 2)])

    def test_reconcile_paginates_and_changes_only_latest(self):
        selected = release('full-v1.7.3', 20)
        api = Mock(side_effect=[[release('desktop-v0.2.1')] * 100, [selected],
                                {'id': 10}, selected, selected])
        self.assertEqual(reconcile_latest(api), selected)
        self.assertEqual(api.call_args_list[1].args, ('releases?per_page=100&page=2',))
        mutations = [call.args for call in api.call_args_list if len(call.args) > 1]
        self.assertEqual(mutations, [('releases/20', {'make_latest': 'true'}, 'PATCH')])

    def test_repeated_reconciliation_is_read_only(self):
        selected = release('full-v1.7.3')
        api = Mock(side_effect=[[selected], selected, selected])
        reconcile_latest(api)
        self.assertTrue(all(len(call.args) == 1 for call in api.call_args_list))

    def test_failed_promotion_never_hides_published_release(self):
        api = Mock(side_effect=[[release('full-v1.7.3', 20)], {'id': 10}, {}, {'id': 10}])
        with self.assertRaisesRegex(ValueError, 'did not persist'):
            reconcile_latest(api)
        self.assertEqual([call.args for call in api.call_args_list if len(call.args) > 1],
                         [('releases/20', {'make_latest': 'true'}, 'PATCH')])
