"""Fetch the pinned previous release for isolated runner upgrade checks."""
import argparse
import hashlib
import json
from pathlib import Path
import urllib.request


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--platform', choices=('macos', 'windows'), required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    baseline = json.loads(Path(__file__).with_name('upgrade-baseline.json').read_text())
    args.output.mkdir(parents=True, exist_ok=True)
    for artifact in baseline['artifacts']:
        if artifact['platform'] != args.platform:
            continue
        target = args.output / artifact['fileName']
        if target.exists():
            raise FileExistsError(target)
        urllib.request.urlretrieve(artifact['url'], target)
        if hashlib.sha256(target.read_bytes()).hexdigest() != artifact['sha256']:
            raise ValueError(f'Upgrade baseline checksum mismatch: {target.name}')


if __name__ == '__main__':
    main()
