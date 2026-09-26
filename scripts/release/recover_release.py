"""Resolve a previously tested component run; never rebuild or move its tag."""
import json
import os
from pathlib import Path
import re
import subprocess


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
    if not re.fullmatch(r'(full|server|desktop)-v\d+\.\d+\.\d+', tag):
        raise ValueError('Invalid component release tag')
    jobs = gh(f'repos/{repo}/actions/runs/{run_id}/jobs?per_page=100')['jobs']
    required = ['Build Windows x64 packages', 'Validate release source and notes',
                'Check the exact release commit / Frontend, Electron, and release scripts',
                'Check the exact release commit / Go test and vet', 'Check the exact release commit / Runtime browser e2e']
    if not tag.startswith('server-'):
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
    attempt = run['run_attempt']
    fields = {'run_id': run_id, 'tag': tag, 'commit': run['head_sha'], 'mode': mode,
              'component': tag.split('-v')[0], 'windows_artifact': f'windows-release-{tag}-{attempt}',
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
