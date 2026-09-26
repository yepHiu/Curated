"""Installer integration checks on the disposable Windows runner; no app is launched."""
import argparse
import os
from pathlib import Path
import subprocess
import tempfile
import winreg
import json
import time
import urllib.request
from release_lib.windows_components import versions
from release_lib.components import artifact_name

ROOT = Path(__file__).resolve().parents[2]
KEY = r'Software\Microsoft\Windows\CurrentVersion\Uninstall\Curated.{}_is1'


def installed(component):
    try:
        with winreg.OpenKey(winreg.HKEY_CURRENT_USER, KEY.format(component.title())) as key:
            return (winreg.QueryValueEx(key, 'DisplayVersion')[0], Path(winreg.QueryValueEx(key, 'InstallLocation')[0]))
    except FileNotFoundError:
        return None


def run(file, *args, success=True):
    result = subprocess.run([str(file), '/VERYSILENT', '/SUPPRESSMSGBOXES', '/SP-', '/NORESTART', '/NOLAUNCH=1', *args])
    if (result.returncode == 0) != success:
        raise RuntimeError(f'{file.name}: unexpected exit {result.returncode}')


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--component', required=True, choices=('server', 'desktop', 'full'))
    args = parser.parse_args()
    current = versions(ROOT)
    selected = ('server', 'desktop') if args.component == 'full' else (args.component,)
    for component in selected:
        if installed(component):
            raise RuntimeError('Smoke checks require a clean runner, refusing to alter an existing installation')
    packages = ROOT / 'release/windows-components'
    setup = lambda c: packages / artifact_name(c, current[c], 'windows', 'x64', 'exe')
    with tempfile.TemporaryDirectory(prefix='curated-installer-smoke-') as temporary:
        base = Path(temporary)
        # Separate data marker proves program uninstall does not remove user data.
        data = base / 'library'
        data.mkdir()
        marker = data / 'keep.txt'
        marker.write_text('existing library')
        os.environ['CURATED_DATA_DIR'] = str(data)
        try:
            if args.component == 'full':
                # Failure after Server succeeds must retain Server and report nonzero.
                desktop_key = KEY.format('desktop'.title())
                with winreg.CreateKey(winreg.HKEY_CURRENT_USER, desktop_key) as key:
                    winreg.SetValueEx(key, 'DisplayVersion', 0, winreg.REG_SZ, 'invalid')
                try:
                    run(setup('full'), success=False)
                    assert installed('server') is not None
                finally:
                    winreg.DeleteKey(winreg.HKEY_CURRENT_USER, desktop_key)
                run(setup('full'))  # retry composes the same Server identity
            else:
                run(setup(args.component), '/DIR=' + str(base / args.component))
            before = {c: installed(c) for c in selected}
            for c, info in before.items():
                assert info and info[0] == current[c], (c, info)
                from release_lib.windows_components import validate_payload
                validate_payload(info[1], c)
            # Start the real Server with a throwaway database and verify Desktop independence.
            server = None
            try:
                if 'server' in selected:
                    config = base / 'server.json'
                    config.write_text(json.dumps({'httpAddr': '127.0.0.1:18881', 'databasePath': str(data / 'curated.db'),
                        'cacheDir': str(data / 'cache'), 'logDir': str(data / 'logs'), 'libraryPaths': []}))
                    server = subprocess.Popen([str(before['server'][1] / 'curated.exe'), '-mode', 'http', '-config', str(config)], cwd=before['server'][1])
                    def health():
                        with urllib.request.urlopen('http://127.0.0.1:18881/api/health', timeout=2) as response:
                            return json.load(response)
                    for attempt in range(60):
                        try:
                            payload = health()
                            assert payload['version'] == current['server']
                            break
                        except Exception:
                            if server.poll() is not None or attempt == 59:
                                raise
                            time.sleep(0.5)
                    with urllib.request.urlopen('http://127.0.0.1:18881/', timeout=5) as response:
                        assert b'<html' in response.read().lower()
                    os.environ['CURATED_SMOKE_SERVER'] = 'http://127.0.0.1:18881'
                if 'desktop' in selected:
                    subprocess.run(['node', str(ROOT / 'scripts/release/desktop_smoke.cjs'), str(before['desktop'][1] / 'Curated Desktop.exe')], check=True, timeout=90)
                if server is not None:
                    assert server.poll() is None, 'Desktop exit stopped independent Server'
                    assert health()['version'] == current['server']
            finally:
                if server is not None:
                    server.terminate()
                    server.wait(timeout=20)
                os.environ.pop('CURATED_SMOKE_SERVER', None)
            if args.component == 'full':
                run(setup('full'))
                assert before == {c: installed(c) for c in selected}
            for c in selected:
                # Standalone upgrade/reinstall preserves the exact Full install location.
                run(setup(c))
                assert installed(c) == before[c]
                with winreg.OpenKey(winreg.HKEY_CURRENT_USER, KEY.format(c.title()), 0, winreg.KEY_SET_VALUE) as key:
                    winreg.SetValueEx(key, 'DisplayVersion', 0, winreg.REG_SZ, '99.0.0')
                run(setup(c), success=False)
                if args.component == 'full':
                    run(setup('full'))
                    assert installed(c)[0] == '99.0.0'
                with winreg.OpenKey(winreg.HKEY_CURRENT_USER, KEY.format(c.title()), 0, winreg.KEY_SET_VALUE) as key:
                    winreg.SetValueEx(key, 'DisplayVersion', 0, winreg.REG_SZ, current[c])
            for c in selected:
                location = installed(c)[1]
                run(location / 'unins000.exe')
                assert installed(c) is None
                assert marker.read_text() == 'existing library'
                for remaining in selected[selected.index(c)+1:]:
                    assert installed(remaining) is not None
        finally:
            for c in selected:
                info = installed(c)
                if info:
                    run(info[1] / 'unins000.exe')
    print('Independent install, reuse, downgrade, partial failure and uninstall checks passed')

if __name__ == '__main__':
    main()
