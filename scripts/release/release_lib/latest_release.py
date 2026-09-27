"""GitHub Latest is a download entry point, not a component update feed."""
import re

VERSION = r'(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)'


def select_latest(releases: list[dict]) -> dict:
    """Prefer the highest public Full SemVer, independent of publication order.

    Legacy releases are a fallback only for repositories with no public Full.
    Component-only releases never become the general-purpose download entry.
    """
    full, legacy = [], []
    for release in releases:
        if release.get('draft', True) or release.get('prerelease', True):
            continue
        tag = release.get('tag_name', '')
        match = re.fullmatch('full-v' + VERSION, tag)
        target = full
        if not match:
            match = re.fullmatch('v' + VERSION, tag)
            target = legacy
        if match:
            target.append((tuple(map(int, match.groups())), release))
    candidates = full or legacy
    if not candidates:
        raise ValueError('No public stable Full or legacy release is available for Latest')
    highest = max(version for version, _ in candidates)
    matches = [release for version, release in candidates if version == highest]
    if len(matches) != 1:
        raise ValueError('Ambiguous releases for the latest version')
    selected = matches[0]
    version = '.'.join(map(str, highest))
    installer = (f'Curated-Full-Setup-{version}-windows-x64.exe' if full
                 else f'Curated-Setup-{version}.exe')
    if sum(asset.get('name') == installer and asset.get('size', 0) > 0
           for asset in selected.get('assets', [])) != 1:
        raise ValueError('Latest candidate is missing its full installer; refusing promotion')
    return selected


def reconcile_latest(api) -> dict:
    """Repair only the Latest flag; never edit tags, assets or public visibility.

    Called after successful publication and on recovery. Listing every page
    prevents a recent single-component release or late older release taking over.
    """
    releases = []
    page = 1
    while True:
        batch = api(f'releases?per_page=100&page={page}')
        releases.extend(batch)
        if len(batch) < 100:
            break
        page += 1
    selected = select_latest(releases)
    if api('releases/latest')['id'] != selected['id']:
        api(f"releases/{selected['id']}", {'make_latest': 'true'}, 'PATCH')
    if api('releases/latest')['id'] != selected['id']:
        raise ValueError('Latest promotion did not persist; retry latest reconciliation')
    return selected


if __name__ == '__main__':
    from scripts.release.cd_release import api
    result = reconcile_latest(api)
    print(f"Latest: {result['tag_name']} ({result['html_url']})")
