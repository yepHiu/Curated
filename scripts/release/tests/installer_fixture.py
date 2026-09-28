"""Build isolated real Inno installers. Never uses production identities or payloads."""
from pathlib import Path
import shutil
import subprocess
import uuid
from unittest.mock import patch

from scripts.release.release_lib import windows_components as windows

ROOT = Path(__file__).resolve().parents[3]


def build_fixture(base: Path) -> dict:
    """Build real installers with disposable identities and harmless application payloads."""
    base = base.resolve()
    base.mkdir(parents=True, exist_ok=False)
    identity = 'Curated.InstallerTest.' + uuid.uuid4().hex[:12]
    helper = base / 'helper.exe'
    fixture = base / 'fixture.exe'
    subprocess.run(['go', 'build', '-ldflags=-s -w', '-o', str(helper), './cmd/curated-migrate'], cwd=ROOT / 'backend', check=True)
    subprocess.run(['go', 'test', '-c', '-o', str(fixture), './internal/installmigration'], cwd=ROOT / 'backend', check=True)
    support = base / 'source/scripts/release/windows'
    support.parent.mkdir(parents=True)
    shutil.copytree(ROOT / 'scripts/release/windows', support)
    template = support / 'Component.iss.tpl'
    source = template.read_text(encoding='utf-8')
    source = source.replace('AppName=Curated __COMPONENT__', 'AppName=Curated Installer Test __COMPONENT__')
    source = source.replace('DefaultDirName={localappdata}\\Programs\\Curated\\__COMPONENT__', 'DefaultDirName=' + str(base / 'installed-__COMPONENT__'))
    source = source.replace('Name: "{autoprograms}\\Curated __COMPONENT__"', 'Name: "{autoprograms}\\' + identity + ' __COMPONENT__"')
    template.write_text(source, encoding='utf-8')
    setups = {}
    for component in ('server', 'desktop'):
        payload = base / ('payload-' + component)
        payload.mkdir()
        exe = 'curated.exe' if component == 'server' else 'Curated Desktop.exe'
        shutil.copy2(fixture, payload / exe)
        shutil.copy2(ROOT / 'icon/curated-desktop.ico', payload / 'curated.ico')
        managed = 'frontend-dist' if component == 'server' else 'resources/app'
        (payload / managed).mkdir(parents=True)
        for version in ('9.0.0', '9.0.1'):
            (payload / managed / 'version.txt').write_text(version)
            values = {'APP_ID': identity + '.' + component, 'SOURCE': str(payload), 'EXE': exe,
                      'PARAMS': '', 'MANAGED_DELETE': managed.replace('/', chr(92)),
                      'LEGACY_CHECK': 'False', 'RUN_FLAGS': 'nowait postinstall skipifsilent',
                      'MIGRATION_HELPER': str(helper)}
            def compile_quietly(command, cwd):
                """Compile the copied template and surface diagnostics only on failure."""
                result = subprocess.run(command[:1] + ['/Q'] + command[1:], cwd=cwd, capture_output=True)
                if result.returncode: raise RuntimeError((result.stdout + result.stderr).decode(errors='replace'))
            with patch.object(windows, '_run', side_effect=compile_quietly):
                setups[(component, version)] = windows.compile_installer(base / 'source', base, base, component, version, values)
    return {'base': base, 'identity': identity, 'helper': helper, 'setups': setups}
