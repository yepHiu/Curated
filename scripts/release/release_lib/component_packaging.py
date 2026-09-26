"""Independent Server/Desktop payloads and an offline Full installer.

This builds local artifacts only. Publishing a multi-EXE release to the legacy
latest-release feed is deliberately not part of this command.
"""
from __future__ import annotations

import json
import os
import re
import shutil
from pathlib import Path

from . import build_steps as steps
from .paths import get_repo_root, resolve_release_path

COMPONENT_IDS = {
    "server": "E973958F-9EBD-47D2-A5CB-58D9F68F7F18",
    "desktop": "19806309-7967-4C86-A2CF-CF106D97B5AE",
}
VARIANTS = ("full", "server", "desktop")
DESKTOP_FILES = ("main.js", "desktop-shell.js", "connections.js", "discovery.js", "updates.js", "settings.js", "preload.cjs", "launcher-preload.cjs")


def validate_version(version: str) -> str:
    """Reject nonnumeric versions before inserting values into installer scripts."""
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        raise ValueError("A numeric major.minor.patch version is required")
    return version


def artifact_name(variant: str, version: str) -> str:
    """Return the exact component/platform name consumed by component updaters."""
    if variant not in VARIANTS:
        raise ValueError("Unknown package variant")
    return f"Curated-{variant.title()}-Setup-{validate_version(version)}-windows-x64.exe"


def stage_components(repo: Path, output: Path, version: str, *, server_binary: Path | None = None,
                     frontend: Path | None = None, electron_runtime: Path | None = None,
                     electron_main: Path | None = None) -> dict[str, Path]:
    """Stage only requested components; refuse to replace an existing build."""
    validate_version(version)
    output.mkdir(parents=True, exist_ok=True)
    result: dict[str, Path] = {}
    icon = repo / "backend/internal/assets/curated.ico"
    if server_binary is not None:
        if frontend is None or not (frontend / "index.html").is_file() or not server_binary.is_file():
            raise FileNotFoundError("Server binary and production Web UI are required")
        destination = output / "server"
        destination.mkdir()
        shutil.copy2(server_binary, destination / "curated.exe")
        shutil.copy2(icon, destination / "curated.ico")
        shutil.copytree(frontend, destination / "frontend-dist")
        steps._bundle_ffmpeg_runtime(repo, destination)
        shutil.copy2(steps._validated_release_library_config_example(repo), destination / "library-config.example.cfg")
        result["server"] = destination
    if electron_runtime is not None:
        required = ("main.js", "preload.cjs", "launcher-preload.cjs", "launcher/index.html")
        if not (electron_runtime / "electron.exe").is_file() or electron_main is None or any(not (electron_main / name).is_file() for name in required):
            raise FileNotFoundError("Windows Electron runtime and built local connection page are required")
        destination = output / "desktop"
        shutil.copytree(electron_runtime, destination)
        (destination / "electron.exe").rename(destination / "Curated.exe")
        app = destination / "resources/app"
        app.mkdir(parents=True, exist_ok=True)
        # Explicit whitelist: no obsolete backend process launcher in the client package.
        main = app / "electron-dist"
        main.mkdir()
        for name in DESKTOP_FILES:
            shutil.copy2(electron_main / name, main / name)
        shutil.copytree(electron_main / "launcher", main / "launcher")
        shutil.copy2(icon, app / "curated.ico")
        shutil.copy2(icon, destination / "curated.ico")
        steps._write_electron_app_package(app, version)
        result["desktop"] = destination
    for component, folder in result.items():
        (folder / "component.json").write_text(json.dumps({"component": component, "version": version, "platform": "windows", "arch": "x64"}, indent=2), encoding="utf-8")
        verify_payload(component, folder)
    return result


def verify_payload(component: str, folder: Path) -> None:
    """Reject incomplete payloads and resources belonging to the other component."""
    files = [p.relative_to(folder).as_posix().lower() for p in folder.rglob("*") if p.is_file()]
    if component == "server":
        if "curated.exe" not in files or "frontend-dist/index.html" not in files or any("electron" in name or name.endswith("resources.pak") for name in files):
            raise ValueError("Server payload contains desktop resources or is incomplete")
    elif component == "desktop":
        if "resources/app/electron-dist/launcher/index.html" not in files or any("frontend-dist/" in name or "ffmpeg" in name or name == "resources/app/curated.exe" or name.endswith(".db") for name in files):
            raise ValueError("Desktop payload contains Server resources or is incomplete")
    else:
        raise ValueError("Unknown component")


def generate_installers(repo: Path, components: dict[str, Path], output: Path, version: str,
                        variant: str = "all") -> list[Path]:
    """Generate standalone scripts first, then a Full wrapper sharing their identities."""
    validate_version(version)
    if variant not in (*VARIANTS, "all"):
        raise ValueError("Unknown package variant")
    output.mkdir(parents=True, exist_ok=True)
    names = ["server", "desktop"] if variant in ("all", "full") else [variant]
    scripts: list[Path] = []
    for component in names:
        payload = components[component]
        verify_payload(component, payload)
        template = (repo / "scripts/release/windows/Component.iss.tpl").read_text(encoding="utf-8")
        values = {"COMPONENT": component, "TITLE": component.title(), "APP_ID": COMPONENT_IDS[component],
                  "VERSION": version, "PAYLOAD": str(payload.resolve()), "OUTPUT": str(output.resolve()),
                  "SETUP_NAME": artifact_name(component, version)[:-4],
                  "EXE": "curated.exe" if component == "server" else "Curated.exe",
                  "ARGS": "-mode tray" if component == "server" else ""}
        for key, value in values.items():
            template = template.replace(f"__{key}__", value)
        script = output / f"Curated-{component}.iss"
        script.write_text(template, encoding="utf-8")
        scripts.append(script)
    if variant in ("all", "full"):
        template = (repo / "scripts/release/windows/Full.iss.tpl").read_text(encoding="utf-8")
        for key, value in {"VERSION": version, "OUTPUT": str(output.resolve()), "SETUP_NAME": artifact_name("full", version)[:-4],
                           "SERVER_APP_ID": COMPONENT_IDS["server"], "DESKTOP_APP_ID": COMPONENT_IDS["desktop"],
                           "SERVER_SETUP": artifact_name("server", version), "DESKTOP_SETUP": artifact_name("desktop", version)}.items():
            template = template.replace(f"__{key}__", value)
        script = output / "Curated-full.iss"
        script.write_text(template, encoding="utf-8")
        scripts.append(script)
    return scripts


def package_components(*, version: str, components_dir: str, output_dir: str, variant: str = "all") -> dict[str, object]:
    """Compile staged payloads when Inno is available; never overwrite existing EXEs."""
    repo = get_repo_root()
    output = resolve_release_path(output_dir, repo)
    root = resolve_release_path(components_dir, repo)
    selected = ("server", "desktop", "full") if variant in ("all", "full") else (variant,)
    for item in selected:
        target = output / artifact_name(item, version)
        if target.exists():
            raise FileExistsError(f"Refusing to overwrite existing installer: {target}")
    scripts = generate_installers(repo, {name: root / name for name in COMPONENT_IDS}, output, version, variant)
    compiler = steps._find_iscc()
    artifacts: list[dict[str, object]] = []
    if compiler:
        for script in scripts:
            component = script.stem.removeprefix("Curated-")
            artifact = output / artifact_name(component, version)
            if artifact.exists():
                raise FileExistsError(f"Refusing to overwrite existing installer: {artifact}")
            steps._run([str(compiler), str(script)], cwd=repo)
            if not artifact.is_file():
                raise FileNotFoundError(f"Compiler did not produce {artifact}")
            artifacts.append({"variant": component, "components": ["server", "desktop"] if component == "full" else [component],
                              "platform": "windows", "arch": "x64", "version": version,
                              "fileName": artifact.name, "sha256": steps._sha256_file(artifact)})
    manifest = {"schemaVersion": 2, "version": version, "status": "built" if compiler else "scripts-only",
                "publicationPolicy": "Do not attach three EXEs to the legacy latest-release feed until update-feed migration is verified.",
                "artifacts": artifacts}
    (output / "components-manifest.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    return {"version": version, "status": manifest["status"], "scripts": [str(p) for p in scripts], "artifacts": artifacts}


def publish_components(*, version: str | None = None, build_stamp: str | None = None, output_dir: str = "release",
                       version_file: str = steps.DEFAULT_VERSION_FILE, history_path: str = steps.DEFAULT_HISTORY_CSV,
                       variant: str = "all") -> dict[str, object]:
    """Allocate one version and build selected local artifacts without publishing them."""
    repo = get_repo_root()
    version_info = steps._resolve_release_version(repo, version, version_file)
    resolved_version = validate_version(str(version_info["version"]))
    stamp = build_stamp or steps.utc_build_stamp()
    root = resolve_release_path(output_dir, repo) / f"components-{resolved_version}-{stamp}"
    if root.exists():
        raise FileExistsError(f"Build directory exists: {root}")
    root.mkdir(parents=True)
    status = "failed"
    result: dict[str, object] = {}
    try:
        needs_server = variant in ("all", "full", "server")
        needs_desktop = variant in ("all", "full", "desktop")
        if not needs_server and not needs_desktop:
            raise ValueError("Unknown package variant")
        binary = frontend = runtime = main = None
        if needs_server:
            frontend = steps.build_frontend(resolved_version, str(root / "frontend"))
            binary = steps.build_backend(resolved_version, stamp, str(root / "backend"))
        if needs_desktop:
            steps._run(["pnpm", "build:electron"], cwd=repo)
            main = repo / "electron-dist"
            runtime = Path(os.environ.get("CURATED_ELECTRON_RUNTIME_DIR", str(repo / "node_modules/electron/dist")))
        stage_components(repo, root / "payloads", resolved_version, server_binary=binary, frontend=frontend,
                         electron_runtime=runtime, electron_main=main)
        result = package_components(version=resolved_version, components_dir=str(root / "payloads"),
                                    output_dir=str(root / "installer"), variant=variant)
        status = "success" if result["status"] == "built" else "partial"
        return result
    finally:
        steps._append_history_row(repo_root=repo, history_path=history_path, version=resolved_version,
                                  build_type=f"release:components:{variant}", artifact_paths=[str(root)], status=status,
                                  operator=steps._release_operator(), notes="Local component build only; Windows installation and legacy migration require validation.")
