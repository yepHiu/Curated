"""Real all-in-one -> Full acceptance. Run only on the disposable Windows CD host."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import sqlite3
import subprocess
import tempfile
import time
import urllib.request
import winreg

from release_lib.windows_components import versions
from release_lib.components import artifact_name
from windows_smoke import installed, run

ROOT = Path(__file__).resolve().parents[2]
LEGACY_KEY = r'Software\Microsoft\Windows\CurrentVersion\Uninstall\{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}_is1'
RUN_KEY = r'Software\Microsoft\Windows\CurrentVersion\Run'


def legacy_location():
    found = []
    for hive in (winreg.HKEY_CURRENT_USER, winreg.HKEY_LOCAL_MACHINE):
        for view in (winreg.KEY_WOW64_32KEY, winreg.KEY_WOW64_64KEY):
            try:
                with winreg.OpenKey(hive, LEGACY_KEY, 0, winreg.KEY_READ | view) as key:
                    found.append(Path(winreg.QueryValueEx(key, 'InstallLocation')[0]))
            except FileNotFoundError:
                pass
    return found


def wait_health(process, version):
    for attempt in range(60):
        try:
            with urllib.request.urlopen('http://127.0.0.1:18882/api/health', timeout=2) as response:
                payload = json.load(response)
                assert version in (payload.get('version'), payload.get('installerVersion'))
            return
        except Exception:
            if process.poll() is not None or attempt == 59:
                raise
            time.sleep(0.5)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--baseline', default='1.5.8')
    args = parser.parse_args()
    migration_root = Path(os.environ['LOCALAPPDATA']) / 'Curated' / 'installer-migrations'
    profile = migration_root.parent / 'server-startup.json'
    if legacy_location() or any(installed(c) for c in ('server', 'desktop')) or migration_root.exists() or profile.exists():
        raise RuntimeError('Legacy acceptance requires a clean disposable host; refusing existing data')
    with winreg.CreateKey(winreg.HKEY_CURRENT_USER, RUN_KEY) as key:
        try:
            winreg.QueryValueEx(key, 'Curated')
        except FileNotFoundError:
            pass
        else:
            raise RuntimeError('Existing startup entry; refusing to overwrite it')
    baseline = next(a for a in json.loads((ROOT / 'scripts/release/legacy-upgrade-baseline.json').read_text())['artifacts'] if a['version'] == args.baseline)
    current = versions(ROOT)
    setup = ROOT / 'release/windows-components' / artifact_name('full', current['full'], 'windows', 'x64', 'exe')
    with tempfile.TemporaryDirectory(prefix='curated-legacy-acceptance-') as temporary:
        base = Path(temporary)
        previous = base / baseline['fileName']
        urllib.request.urlretrieve(baseline['url'], previous)
        assert hashlib.sha256(previous.read_bytes()).hexdigest() == baseline['sha256']
        data = base / 'external data'
        data.mkdir()
        config = base / 'old server config.json'
        database = data / 'original.db'
        config.write_text(json.dumps({'httpAddr': '127.0.0.1:18882', 'databasePath': str(database),
                                     'cacheDir': str(data / 'cache'), 'logDir': str(data / 'logs')}))
        environment = os.environ.copy()
        environment['CURATED_DATA_DIR'] = str(data)
        environment.pop('CURATED_HOSTED_BY', None)
        try:
            run(previous, '/DIR=' + str(base / 'old application'))
            old = legacy_location()[0]
            prior = subprocess.Popen([str(old / 'resources/app/curated.exe'), '-mode', 'http', '-config', str(config)],
                                     cwd=old / 'resources/app', env=environment)
            try:
                wait_health(prior, baseline['version'])
            finally:
                prior.terminate()
                prior.wait(timeout=20)
            with sqlite3.connect(database) as db:
                db.execute('CREATE TABLE full_upgrade_marker (favorite INTEGER, rating INTEGER)')
                db.execute('INSERT INTO full_upgrade_marker VALUES (1, 5)')
            settings = data / 'config/library-config.cfg'
            settings.parent.mkdir(parents=True, exist_ok=True)
            settings.write_text(json.dumps({'launchAtLogin': True, 'autoLibraryWatch': False}))
            with winreg.CreateKey(winreg.HKEY_CURRENT_USER, RUN_KEY) as key:
                winreg.SetValueEx(key, 'Curated', 0, winreg.REG_SZ,
                                 f'"{old / "resources/app/curated.exe"}" -mode tray -autostart -config "{config}"')
            # A missing custom data directory must not silently create an empty library.
            run(setup, '/LEGACYDATADIR=' + str(base / 'missing'), success=False)
            assert legacy_location() and not installed('server')
            assert database.is_file()
            run(setup, '/LEGACYDATADIR=' + str(data), '/LEGACYCONFIG=' + str(config))
            assert not legacy_location(), 'Old registration survived Full migration'
            for component in ('server', 'desktop'):
                assert installed(component)[0] == current[component]
            journal = json.loads((migration_root / 'legacy.json').read_text())
            assert journal['stage'] == 'complete'
            assert Path(journal['backup']).is_file()
            assert json.loads(profile.read_text())['databasePath'] == str(database)
            with winreg.OpenKey(winreg.HKEY_CURRENT_USER, RUN_KEY) as key:
                command = winreg.QueryValueEx(key, 'Curated')[0]
                assert str(installed('server')[1] / 'curated.exe') in command
                assert str(old) not in command
            # Launch from the ordinary shortcut command: no custom environment/config.
            clean_environment = os.environ.copy()
            clean_environment.pop('CURATED_DATA_DIR', None)
            server = subprocess.Popen([str(installed('server')[1] / 'curated.exe'), '-mode', 'http'],
                                      cwd=installed('server')[1], env=clean_environment)
            try:
                wait_health(server, current['server'])
                with urllib.request.urlopen('http://127.0.0.1:18882/', timeout=5) as response:
                    assert b'<html' in response.read().lower()
                with sqlite3.connect(database) as db:
                    assert db.execute('SELECT favorite, rating FROM full_upgrade_marker').fetchone() == (1, 5)
                    assert db.execute('PRAGMA integrity_check').fetchone() == ('ok',)
            finally:
                server.terminate()
                server.wait(timeout=20)
            assert json.loads(settings.read_text())['launchAtLogin'] is True
            run(setup)  # completed migration is idempotent
            assert json.loads((migration_root / 'legacy.json').read_text())['backup'] == journal['backup']
            for component in ('server', 'desktop'):
                run(installed(component)[1] / 'unins000.exe')
            assert database.is_file() and profile.is_file() and Path(journal['backup']).is_file()
        finally:
            for component in ('server', 'desktop'):
                info = installed(component)
                if info:
                    run(info[1] / 'unins000.exe')
            for location in set(legacy_location()):
                run(location / 'unins000.exe')
            # Only fixtures created after the clean-host guard above are removed.
            with winreg.CreateKey(winreg.HKEY_CURRENT_USER, RUN_KEY) as key:
                try:
                    winreg.DeleteValue(key, 'Curated')
                except FileNotFoundError:
                    pass
            profile.unlink(missing_ok=True)
            if migration_root.exists():
                shutil.rmtree(migration_root)
    print('Real legacy installation -> Full, custom data/config, startup and preserved database passed')


if __name__ == '__main__':
    main()
