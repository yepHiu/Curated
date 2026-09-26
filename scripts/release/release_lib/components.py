"""Split release identities. Planning is read-only; legacy packagers stay isolated."""
from pathlib import Path
import re

from .versioning import format_version, read_version_state


COMPONENTS = ("desktop", "server", "full")


def artifact_name(component: str, version: str, platform: str, arch: str, format: str) -> str:
    if component not in COMPONENTS or not re.fullmatch(r"(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)", version):
        raise ValueError("Expected a release component and numeric SemVer.")
    supported_formats = {"windows": ("exe", "zip"), "macos": ("dmg", "pkg"), "linux": ("tar.gz",)}
    if platform not in supported_formats or format not in supported_formats[platform]:
        raise ValueError("Unsupported platform / format combination.")
    if arch not in ("x64", "arm64") and not (platform == "macos" and arch == "universal"):
        raise ValueError("Unsupported platform / architecture combination.")
    if platform == "macos" and ((format == "dmg") != (component == "desktop")):
        raise ValueError("macOS Desktop uses DMG; Server / Full reserve PKG.")
    setup = "-Setup" if format in ("exe", "pkg") else ""
    return f"Curated-{component.title()}{setup}-{version}-{platform}-{arch}.{format}"


def component_plan(repo_root: Path, component: str, platform: str, arch: str, format: str) -> dict:
    if component not in COMPONENTS:
        raise ValueError("Unknown component.")
    def version_of(name: str) -> str:
        return format_version(read_version_state(repo_root / "scripts/release/versions" / f"{name}.json")["current"])

    version = version_of(component)
    result = {
        "schema": 1, "component": component, "variant": "standalone" if component != "full" else "bundle",
        "version": version, "platform": platform, "arch": arch, "format": format,
        "fileName": artifact_name(component, version, platform, arch, format),
        "tag": f"{component}-v{version}", "status": "planned",
    }
    if component == "full":
        result["components"] = {"server": version_of("server"), "desktop": version_of("desktop")}
    return result
