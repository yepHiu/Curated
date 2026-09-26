"""Native macOS component installers. No installation, signing credentials or upload."""
from __future__ import annotations

import json
import platform
import plistlib
import shutil
import subprocess
from pathlib import Path
from xml.etree import ElementTree as ET

from . import build_steps as steps
from .component_packaging import DESKTOP_FILES, validate_version
from .macos_runtime import brand_desktop
from .paths import get_repo_root, resolve_release_path


def artifact_name(component: str, version: str, arch: str) -> str:
    """Generate a validated component/version/CPU filename."""
    if component != "desktop" or arch != "arm64":
        raise ValueError("Unsupported macOS package")
    return f"Curated-{component.title()}-Setup-{validate_version(version)}-macos-{arch}.pkg"


def write_distribution(target: Path, components: list[str], version: str, arch: str) -> None:
    """Describe the sole supported Apple Silicon Desktop installer."""
    if components != ["desktop"] or arch != "arm64":
        raise ValueError("macOS supports Apple Silicon Desktop only")
    root = ET.Element("installer-gui-script", minSpecVersion="2")
    ET.SubElement(root, "title").text = "Curated"
    ET.SubElement(root, "options", customize="never", requireScripts="false", hostArchitectures="arm64")
    ET.SubElement(root, "domains", enable_anywhere="false", enable_currentUserHome="false", enable_localSystem="true")
    volume = ET.SubElement(root, "volume-check")
    allowed = ET.SubElement(volume, "allowed-os-versions")
    ET.SubElement(allowed, "os-version", min="15.0")
    choices = ET.SubElement(root, "choices-outline")
    for component in components:
        identifier = f"app.curated.{component}"
        ET.SubElement(choices, "line", choice=component)
        choice = ET.SubElement(root, "choice", id=component, title=f"Curated {component.title()}", visible="false", start_selected="true")
        ET.SubElement(choice, "pkg-ref", id=identifier)
        ET.SubElement(root, "pkg-ref", id=identifier, version=version).text = f"{component}.pkg"
    ET.indent(root)
    ET.ElementTree(root).write(target, encoding="utf-8", xml_declaration=True)


def create_icon(repo: Path, root: Path) -> Path:
    """Generate an Apple iconset from the existing project PNG."""
    iconset = root / "curated.iconset"
    iconset.mkdir()
    for size in (16, 32, 128, 256, 512):
        for scale in (1, 2):
            filename = f"icon_{size}x{size}{'@2x' if scale == 2 else ''}.png"
            subprocess.run(["sips", "-z", str(size * scale), str(size * scale), str(repo / "public/Curated-icon.png"), "--out", str(iconset / filename)], check=True, stdout=subprocess.DEVNULL)
    icon = root / "curated.icns"
    subprocess.run(["iconutil", "-c", "icns", str(iconset), "-o", str(icon)], check=True)
    return icon


def stage_desktop(repo: Path, root: Path, version: str, arch: str) -> None:
    """Stage the client-only app bundle without any Server files."""
    bundle = root / "Applications/Curated.app"
    bundle.parent.mkdir(parents=True)
    shutil.copytree(repo / "node_modules/electron/dist/Electron.app", bundle, symlinks=True)
    resources = bundle / "Contents/Resources"
    (resources / "default_app.asar").unlink(missing_ok=True)
    app = resources / "app"
    main = app / "electron-dist"
    main.mkdir(parents=True)
    for name in DESKTOP_FILES:
        shutil.copy2(repo / "electron-dist" / name, main / name)
    shutil.copytree(repo / "electron-dist/launcher", main / "launcher")
    shutil.copy2(repo / "public/Curated-icon.png", app / "curated.png")
    steps._write_electron_app_package(app, version)
    brand_desktop(bundle, version, arch, create_icon(repo, root.parent))


def publish_macos(*, version: str | None = None, build_stamp: str | None = None, output_dir: str = "release",
                  version_file: str = steps.DEFAULT_VERSION_FILE, history_path: str = steps.DEFAULT_HISTORY_CSV,
                  variant: str = "all", arch: str = "arm64") -> dict[str, object]:
    """Build native component products under a fresh directory and record results."""
    if arch != "arm64" or variant not in ("all", "desktop"):
        raise ValueError("macOS supports Apple Silicon Desktop only")
    variant = "desktop"
    if platform.system() != "Darwin" or platform.machine() != "arm64":
        raise ValueError("macOS packages must be built on a native runner matching the requested architecture")
    repo = get_repo_root()
    resolved = validate_version(str(steps._resolve_release_version(repo, version, version_file)["version"]))
    stamp = build_stamp or steps.utc_build_stamp()
    root = resolve_release_path(output_dir, repo) / f"components-{resolved}-{stamp}-macos-{arch}"
    root.mkdir(parents=True, exist_ok=False)
    installer = root / "installer"
    installer.mkdir()
    packages = root / "packages"
    packages.mkdir()
    components = ["desktop"]
    status = "failed"
    try:
        for component in components:
            payload = root / "payloads" / component
            steps._run(["pnpm", "build:electron"], cwd=repo)
            stage_desktop(repo, payload, resolved, arch)
            # Do not relocate a Desktop update to an unrelated developer bundle.
            component_plist = packages / f"{component}.plist"
            steps._run(["pkgbuild", "--analyze", "--root", str(payload), str(component_plist)], cwd=repo)
            with component_plist.open("rb") as stream:
                configuration = plistlib.load(stream)
            for item in configuration:
                item.update(BundleIsRelocatable=False, BundleOverwriteAction="upgrade")
            with component_plist.open("wb") as stream:
                plistlib.dump(configuration, stream)
            steps._run(["pkgbuild", "--root", str(payload), "--component-plist", str(component_plist), "--identifier", f"app.curated.{component}", "--version", resolved, "--install-location", "/", str(packages / f"{component}.pkg")], cwd=repo)
        artifacts = []
        for component in components:
            selected = ["desktop"]
            distribution = packages / f"{component}.xml"
            write_distribution(distribution, selected, resolved, arch)
            artifact = installer / artifact_name(component, resolved, arch)
            steps._run(["productbuild", "--distribution", str(distribution), "--package-path", str(packages), str(artifact)], cwd=repo)
            artifacts.append({"variant": component, "components": selected, "platform": "macos", "arch": arch,
                              "version": resolved, "fileName": artifact.name, "sha256": steps._sha256_file(artifact)})
        manifest = {"schemaVersion": 2, "version": resolved, "status": "built", "signing": "ad-hoc-app; unsigned-installer; not-notarized",
                    "minimumOSVersion": "15.0", "publicationPolicy": "CI test artifacts only; no legacy latest-feed publication.", "artifacts": artifacts}
        (installer / "components-manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
        status = "success"
        return manifest
    finally:
        steps._append_history_row(repo_root=repo, history_path=history_path, version=resolved, build_type=f"release:macos:{arch}:{variant}",
                                  artifact_paths=[str(root)], status=status, operator=steps._release_operator(), notes="Unsigned test installers; no publication or installation performed.")
