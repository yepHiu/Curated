"""Resolve a previously tested component run; never rebuild or move its tag."""
import json
import os
from pathlib import Path
import re
import subprocess
import base64


def gh(path: str, raw=False):
    output = subprocess.check_output(['gh', 'api', path])
    return output.decode('utf-8', errors='replace') if raw else json.loads(output)


def main():
    request = json.loads(Path('scripts/release/recovery-request.json').read_text())
    run_id = os.environ.get('REQUEST_RUN') or str(request['runId'])
    mode = os.environ.get('REQUEST_MODE') or request['mode']
    if not run_id.isdigit() or mode not in ('draft', 'publish'):
        raise ValueError('Invalid recovery request')
    repo = os.environ['GITHUB_REPOSITORY']
    run = gh(f'repos/{repo}/actions/runs/{run_id}')
    if run['path'] != '.github/workflows/cd-release.yml' or run['conclusion'] != 'failure' or run['event'] != 'push':
        raise ValueError('Recovery only accepts a failed component tag CD run')
    tag = run['head_branch']
    if not re.fullmatch(r'(?:(full|server|desktop)-v\d+\.\d+\.\d+|release-\d{8}(?:-(?:[2-9]|[1-9]\d+))?)', tag):
        raise ValueError('Invalid component release tag')
    # A Server-prefixed bridge tag can contain both components. Recover the exact
    # selection from the immutable source, never infer it from that tag prefix.
    component = tag.split('-v')[0]
    tree = gh(f"repos/{repo}/git/trees/{run['head_sha']}?recursive=1")
    if tree.get('truncated'):
        raise ValueError('Cannot resolve recovery batch from a truncated source tree')
    path = f'scripts/release/batches/{tag}.json'
    entry = next((e for e in tree['tree'] if e['path'] == path), None)
    if entry:
        blob = gh(f"repos/{repo}/git/blobs/{entry['sha']}")
        batch = json.loads(base64.b64decode(blob['content']))
        if batch['tag'] != tag or batch['schema'] != 1:
            raise ValueError('Invalid recovery batch')
        selected = [c for c in ('server', 'desktop') if batch['modules'][c]['changed']]
        if not selected:
            raise ValueError('Empty recovery batch')
        component = 'both' if len(selected) == 2 else selected[0]
    elif tag.startswith('release-'):
        raise ValueError('Missing recovery batch')
    jobs = gh(f'repos/{repo}/actions/runs/{run_id}/jobs?per_page=100')['jobs']
    required = ['Build Windows x64 packages', 'Validate release source and notes',
                'Check the exact release commit / Frontend, Electron, and release scripts',
                'Check the exact release commit / Go test and vet', 'Check the exact release commit / Runtime browser e2e']
    if component != 'server':
        required.append('Build Mac Desktop (Apple Silicon)')
    for name in required:
        if not any(j['name'] == name and j['conclusion'] == 'success' for j in jobs):
            raise ValueError(f'Required gate did not pass: {name}')
    for job in jobs:
        if job['conclusion'] == 'failure':
            try:
                logs = gh(f"repos/{repo}/actions/jobs/{job['id']}/logs", raw=True)
            except subprocess.CalledProcessError:
                print('::warning::Previous log download unavailable; continuing artifact verification')
                continue
            # Only error/traceback vicinity, capped to avoid exposing verbose job logs.
            lines = logs.splitlines()
            indexes = [i for i, line in enumerate(lines) if any(word in line for word in ('Traceback', 'Error:', 'ValueError', 'HTTPError', 'Exception'))]
            selected = sorted({i for index in indexes for i in range(max(0, index-2), min(len(lines), index+8))})
            for i in selected[-35:]:
                line = lines[i].replace('%', '%25').replace('\r', '%0D').replace('\n', '%0A')
                print('::warning::Previous CD: ' + line)
    releases = []
    page = 1
    while True:
        items = gh(f'repos/{repo}/releases?per_page=100&page={page}')
        releases.extend(items)
        if len(items) < 100:
            break
        page += 1
    published = next((release for release in releases if release['tag_name'] == tag and not release['draft']), None)
    command = 'channels' if published else 'publish'
    attempt = run['run_attempt']
    fields = {'command': command, 'run_id': run_id, 'tag': tag, 'commit': run['head_sha'], 'mode': mode,
              'component': component, 'windows_artifact': f'windows-release-{tag}-{attempt}',
              'macos_artifact': f'macos-desktop-{tag}-{attempt}'}
    with open(os.environ['GITHUB_OUTPUT'], 'a') as stream:
        stream.writelines(f'{key}={value}\n' for key, value in fields.items())


if __name__ == '__main__':
    try:
        main()
    except Exception as error:
        import traceback
        detail = traceback.format_exc()[-6000:].replace('%', '%25').replace('\r', '%0D').replace('\n', '%0A')
        print('::error title=Release diagnostics::' + detail, flush=True)
        raise
