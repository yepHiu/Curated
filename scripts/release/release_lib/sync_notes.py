"""Synchronize published component descriptions without changing release artifacts."""
import os
from pathlib import Path
import re
import subprocess

from scripts.release import cd_release, component_cd

ROOT = Path(__file__).resolve().parents[3]


def snapshot(release):
    fields = ('id', 'tag_name', 'target_commitish', 'name', 'draft', 'prerelease', 'published_at')
    return ({key: release[key] for key in fields},
            sorted((a['id'], a['name'], a['size'], a.get('digest'), a['browser_download_url']) for a in release['assets']))


def main():
    requested = os.environ.get('RELEASE_TAG', '')
    if requested:
        tags = [requested]
    else:
        before = os.environ['BEFORE_SHA']
        after = os.environ['GITHUB_SHA']
        if not re.fullmatch(r'[a-f0-9]{40}', before) or not re.fullmatch(r'[a-f0-9]{40}', after):
            raise ValueError('Invalid source range')
        files = subprocess.check_output(['git', 'diff', '--name-only', '--diff-filter=AM', before, after, '--', 'docs/release-notes/'], cwd=ROOT, text=True).splitlines()
        tags = [Path(file).stem for file in files if component_cd.PATTERN.fullmatch(Path(file).stem)]
    latest_id = cd_release.api('releases/latest')['id']
    for tag in tags:
        if not component_cd.PATTERN.fullmatch(tag):
            raise ValueError('Expected a component release tag')
        release = cd_release.api(f'releases/tags/{tag}')
        if release is None or release['draft']:
            print(f'Skip unpublished notes: {tag}')
            continue
        commit = cd_release.git(ROOT, 'rev-parse', '--verify', f'refs/tags/{tag}^{{commit}}')
        marker = cd_release.source_marker({'commit': commit})
        if marker not in release['body']:
            raise ValueError(f'Original source marker does not match tag: {tag}')
        body = component_cd.body(ROOT, {'tag': tag, 'commit': commit})
        urls = re.findall(r'https://github.com/[^\s)]+/releases/download/[^\s)]+', body)
        available = {a['browser_download_url'] for a in release['assets']}
        if not urls or not set(urls) <= available:
            raise ValueError(f'Notes link to unavailable release assets: {tag}')
        original = snapshot(release)
        cd_release.api(f"releases/{release['id']}", {'body': body}, 'PATCH')
        updated = cd_release.api(f"releases/{release['id']}")
        if updated['body'] != body or snapshot(updated) != original:
            raise ValueError(f'Release verification failed: {tag}')
        if cd_release.api('releases/latest')['id'] != latest_id:
            raise ValueError('Latest changed during description update')
        print(f'Updated and verified: {updated["html_url"]}')


if __name__ == '__main__':
    main()
