"""Download the pinned QA engine. This does not prepare a distributable license/source bundle."""
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import urllib.request

root = Path(__file__).resolve().parents[2]
pin = json.loads((root / 'scripts/release/native-player/windows-x64.json').read_text())
output = root / '.workspace/native-player'
output.mkdir(parents=True, exist_ok=True)
archive = output / 'engine.7z'
if not archive.exists() or hashlib.sha256(archive.read_bytes()).hexdigest() != pin['artifactSha256']:
    urllib.request.urlretrieve(pin['artifactUrl'], archive)
if hashlib.sha256(archive.read_bytes()).hexdigest() != pin['artifactSha256']:
    raise RuntimeError('Pinned engine archive checksum mismatch')
extractor = shutil.which('7z') or shutil.which('7z.exe')
if not extractor:
    raise RuntimeError('7-Zip is required to extract the development engine')
subprocess.run([extractor, 'x', str(archive), '-o' + str(output), '-y', 'mpv.exe'], check=True, stdout=subprocess.DEVNULL)
if hashlib.sha256((output / 'mpv.exe').read_bytes()).hexdigest() != pin['executableSha256']:
    raise RuntimeError('Pinned mpv checksum mismatch')
(output / 'engine-pin.json').write_text(json.dumps(pin, indent=2) + '\n')
print('Pinned Windows QA engine ready: ' + str(output / 'mpv.exe'))
print('Production bundles additionally require corresponding sources, build inputs and license materials.')
