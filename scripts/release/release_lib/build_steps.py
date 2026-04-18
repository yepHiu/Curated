from __future__ import annotations

import hashlib
import json
import os
import shutil
import subprocess
from datetime import datetime, timezone
from pathlib import Path
from zipfile import ZIP_DEFLATED, ZipFile

from .git_utils import current_branch, current_short_commit, load_git_log, resolve_commit
from .history import (
    append_history_entry,
    build_package_history_change_summary,
    migrate_markdown_history_to_csv,
)
from .paths import get_repo_root, resolve_release_path, to_repo_relative_path
from .versioning import allocate_next_patch_in_file, format_version, read_version_state, set_version_base_in_file


DEFAULT_VERSION_FILE = "scripts/release/version.json"
DEFAULT_HISTORY_CSV = "docs/package-build-history.csv"
LEGACY_HISTORY_MD = "docs/2026-04-02-package-build-history.md"


def utc_build_stamp() -> str:
    return datetime.now(timezone.utc).strftime("%Y%m%d.%H%M%S")


def show_version(version_file: str = DEFAULT_VERSION_FILE) -> dict[str, object]:
    repo_root = get_repo_root()
    state = read_version_state(resolve_release_path(version_file, repo_root))
    return {"state": state, "version": format_version(state["current"])}


def set_version_base(major: int, minor: int, version_file: str = DEFAULT_VERSION_FILE) -> dict[str, object]:
    repo_root = get_repo_root()
    result = set_version_base_in_file(resolve_release_path(version_file, repo_root), major, minor)
    return {"version": result["version"], "source": "manual-base", "state": result["state"]}


def build_frontend(version: str, output_dir: str = "release/frontend") -> Path:
    repo_root = get_repo_root()
    resolved_output_dir = resolve_release_path(output_dir, repo_root)

    print("==> Building frontend dist")
    print(f"Version: {version}")
    print(f"Output : {resolved_output_dir}")

    env = os.environ.copy()
    env["VITE_APP_VERSION"] = version

    _run(["pnpm", "typecheck"], cwd=repo_root, env=env)
    _run(["pnpm", "exec", "vite", "build", "--configLoader", "native"], cwd=repo_root, env=env)

    if resolved_output_dir.exists():
        shutil.rmtree(resolved_output_dir)
    resolved_output_dir.mkdir(parents=True, exist_ok=True)
    dist_dir = repo_root / "dist"
    for child in dist_dir.iterdir():
        destination = resolved_output_dir / child.name
        if child.is_dir():
            shutil.copytree(child, destination, dirs_exist_ok=True)
        else:
            shutil.copy2(child, destination)

    print("Frontend build complete.")
    return resolved_output_dir


def build_backend(
    version: str,
    build_stamp: str,
    output_dir: str = "release/backend",
    binary_name: str = "curated.exe",
) -> Path:
    repo_root = get_repo_root()
    backend_root = repo_root / "backend"
    go_cache_dir = repo_root / ".gocache"
    go_tmp_dir = repo_root / ".tmp-go"
    resolved_output_dir = resolve_release_path(output_dir, repo_root)
    binary_path = resolved_output_dir / binary_name

    print("==> Building backend release binary")
    print(f"Version   : {version}")
    print(f"BuildStamp: {build_stamp}")
    print(f"Output    : {binary_path}")

    if resolved_output_dir.exists():
        shutil.rmtree(resolved_output_dir)
    resolved_output_dir.mkdir(parents=True, exist_ok=True)
    go_cache_dir.mkdir(parents=True, exist_ok=True)
    go_tmp_dir.mkdir(parents=True, exist_ok=True)

    env = os.environ.copy()
    env["GOCACHE"] = str(go_cache_dir)
    env["GOTMPDIR"] = str(go_tmp_dir)
    env["GOTELEMETRY"] = "off"
    ldflags = (
        "-H=windowsgui "
        f"-X curated-backend/internal/version.BuildStamp={build_stamp} "
        f"-X curated-backend/internal/version.InstallerVersion={version}"
    )
    _run(
        ["go", "build", "-tags", "release", "-ldflags", ldflags, "-o", str(binary_path), "./cmd/curated"],
        cwd=backend_root,
        env=env,
    )

    print("Backend release build complete.")
    return binary_path


def assemble_release(
    version: str,
    build_stamp: str,
    binary_path: str = "release/backend/curated.exe",
    frontend_dist_dir: str = "release/frontend",
    output_dir: str = "release/Curated",
) -> Path:
    repo_root = get_repo_root()
    resolved_binary_path = resolve_release_path(binary_path, repo_root)
    resolved_frontend_dist_dir = resolve_release_path(frontend_dist_dir, repo_root)
    resolved_output_dir = resolve_release_path(output_dir, repo_root)
    runtime_dir = resolved_output_dir / "runtime"
    docs_dir = resolved_output_dir / "docs"
    third_party_dir = repo_root / "backend" / "third_party"

    if not resolved_binary_path.exists():
        raise FileNotFoundError(f"Release binary not found: {resolved_binary_path}")

    print("==> Assembling release directory")
    print(f"Version : {version}")
    print("Channel : release")
    print(f"Output  : {resolved_output_dir}")

    if resolved_output_dir.exists():
        shutil.rmtree(resolved_output_dir)

    (runtime_dir / "config").mkdir(parents=True, exist_ok=True)
    (runtime_dir / "data").mkdir(parents=True, exist_ok=True)
    (runtime_dir / "cache").mkdir(parents=True, exist_ok=True)
    (runtime_dir / "logs").mkdir(parents=True, exist_ok=True)
    docs_dir.mkdir(parents=True, exist_ok=True)

    shutil.copy2(resolved_binary_path, resolved_output_dir / "curated.exe")
    shutil.copy2(repo_root / "backend" / "internal" / "assets" / "curated.ico", resolved_output_dir / "curated.ico")

    if resolved_frontend_dist_dir.exists():
        shutil.copytree(
            resolved_frontend_dist_dir,
            resolved_output_dir / "frontend-dist",
            dirs_exist_ok=True,
        )

    if third_party_dir.exists():
        shutil.copytree(third_party_dir, resolved_output_dir / "third_party", dirs_exist_ok=True)

    shutil.copy2(
        repo_root / "config" / "library-config.cfg",
        runtime_dir / "config" / "library-config.example.cfg",
    )
    shutil.copy2(
        repo_root / "docs" / "plan" / "2026-03-31-production-packaging-and-config-strategy.md",
        docs_dir / "production-packaging-and-config-strategy.md",
    )

    notes = (
        "Curated release package\n\n"
        f"Version    : {version}\n"
        f"BuildStamp : {build_stamp}\n"
        "Channel    : release\n\n"
        "This package is prepared for both installer and portable distribution.\n\n"
        "Runtime data layout:\n"
        "  config\\\n"
        "  data\\\n"
        "  cache\\\n"
        "  logs\\\n\n"
        "Current status:\n"
        "  - curated.exe is the release backend binary.\n"
        "  - frontend-dist contains the production frontend output.\n"
        "  - third_party can contain bundled runtime tools such as ffmpeg.\n"
        "  - runtime\\config\\library-config.example.cfg is a sample library settings file.\n\n"
        "Target production behavior:\n"
        "  - release builds should use the per-user data directory by default.\n"
        "  - config, database, cache, and logs should stay outside the install directory.\n"
    )
    (resolved_output_dir / "README-release.txt").write_text(notes, encoding="utf-8")

    print("Release directory assembled.")
    return resolved_output_dir


def package_portable(
    version: str | None = None,
    input_dir: str = "release/Curated",
    output_dir: str = "release/portable",
    version_file: str = DEFAULT_VERSION_FILE,
    history_path: str = DEFAULT_HISTORY_CSV,
    skip_history: bool = False,
) -> dict[str, object]:
    repo_root = get_repo_root()
    version_info = _resolve_release_version(repo_root, version, version_file)
    resolved_input_dir = resolve_release_path(input_dir, repo_root)
    resolved_output_dir = resolve_release_path(output_dir, repo_root)
    zip_path = resolved_output_dir / f"Curated-{version_info['version']}-windows-x64.zip"
    status = "failed"
    artifact_paths: list[str] = []
    notes = f"versionSource={version_info['source']}"

    try:
        if not resolved_input_dir.exists():
            raise FileNotFoundError(f"Release directory not found: {resolved_input_dir}")

        print("==> Packaging portable zip")
        print(f"Version: {version_info['version']}")
        print(f"Source : {version_info['source']}")
        print(f"Input  : {resolved_input_dir}")
        print(f"Output : {zip_path}")

        resolved_output_dir.mkdir(parents=True, exist_ok=True)
        if zip_path.exists():
            zip_path.unlink()
        _zip_directory_contents(resolved_input_dir, zip_path)

        status = "success"
        artifact_paths = [str(zip_path)]
        print("Portable zip created.")
        return {
            "version": version_info["version"],
            "status": status,
            "artifact_paths": artifact_paths,
            "notes": notes,
        }
    except Exception as exc:
        notes = f"{notes}; error={exc}"
        raise
    finally:
        if not skip_history and version_info["version"]:
            _append_history_row(
                repo_root=repo_root,
                history_path=history_path,
                version=version_info["version"],
                build_type="release:portable",
                artifact_paths=artifact_paths,
                status=status,
                operator=_release_operator(),
                notes=notes,
            )


def package_installer(
    version: str | None = None,
    app_dir: str = "release/Curated",
    output_dir: str = "release/installer",
    template_path: str = "scripts/release/windows/Curated.iss.tpl",
    version_file: str = DEFAULT_VERSION_FILE,
    history_path: str = DEFAULT_HISTORY_CSV,
    skip_history: bool = False,
) -> dict[str, object]:
    repo_root = get_repo_root()
    version_info = _resolve_release_version(repo_root, version, version_file)
    resolved_app_dir = resolve_release_path(app_dir, repo_root)
    resolved_output_dir = resolve_release_path(output_dir, repo_root)
    resolved_template_path = resolve_release_path(template_path, repo_root)
    generated_iss_path = resolved_output_dir / "Curated.iss"
    setup_base_name = f"Curated-Setup-{version_info['version']}"
    status = "failed"
    artifact_paths: list[str] = []
    notes = f"versionSource={version_info['source']}"

    try:
        if not resolved_app_dir.exists():
            raise FileNotFoundError(f"Release directory not found: {resolved_app_dir}")
        if not resolved_template_path.exists():
            raise FileNotFoundError(f"Installer template not found: {resolved_template_path}")

        print("==> Packaging installer")
        print(f"Version: {version_info['version']}")
        print(f"Source : {version_info['source']}")
        print(f"App dir: {resolved_app_dir}")
        print(f"Output : {resolved_output_dir}")

        resolved_output_dir.mkdir(parents=True, exist_ok=True)
        template = resolved_template_path.read_text(encoding="utf-8")
        template = template.replace("__APP_VERSION__", version_info["version"])
        template = template.replace("__APP_DIR__", str(resolved_app_dir).replace("\\", "\\\\"))
        template = template.replace("__OUTPUT_DIR__", str(resolved_output_dir).replace("\\", "\\\\"))
        template = template.replace("__SETUP_BASENAME__", setup_base_name)
        generated_iss_path.write_text(template, encoding="utf-8")

        iscc_path = _find_iscc()
        if not iscc_path:
            print(f"WARNING: ISCC.exe not found. Generated installer script only: {generated_iss_path}")
            print("WARNING: Install Inno Setup and rerun this command to build the installer.")
            status = "partial"
            artifact_paths = [str(generated_iss_path)]
            notes = f"{notes}; ISCC.exe not found, installer executable not generated"
            return {
                "version": version_info["version"],
                "status": status,
                "artifact_paths": artifact_paths,
                "notes": notes,
            }

        _run([str(iscc_path), str(generated_iss_path)], cwd=repo_root)

        status = "success"
        artifact_paths = [str(resolved_output_dir / f"{setup_base_name}.exe")]
        print("Installer build complete.")
        return {
            "version": version_info["version"],
            "status": status,
            "artifact_paths": artifact_paths,
            "notes": notes,
        }
    except Exception as exc:
        notes = f"{notes}; error={exc}"
        raise
    finally:
        if not skip_history and version_info["version"]:
            _append_history_row(
                repo_root=repo_root,
                history_path=history_path,
                version=version_info["version"],
                build_type="release:installer",
                artifact_paths=artifact_paths,
                status=status,
                operator=_release_operator(),
                notes=notes,
            )


def publish_release(
    version: str | None = None,
    build_stamp: str | None = None,
    output_dir: str = "release",
    version_file: str = DEFAULT_VERSION_FILE,
    history_path: str = DEFAULT_HISTORY_CSV,
) -> dict[str, object]:
    repo_root = get_repo_root()
    version_info = _resolve_release_version(repo_root, version, version_file)
    resolved_output_dir = resolve_release_path(output_dir, repo_root)
    build_stamp_value = build_stamp or utc_build_stamp()
    manifest_dir = resolved_output_dir / "manifest"
    portable_zip = resolved_output_dir / "portable" / f"Curated-{version_info['version']}-windows-x64.zip"
    installer_exe = resolved_output_dir / "installer" / f"Curated-Setup-{version_info['version']}.exe"
    assembled_dir = resolved_output_dir / "Curated"
    binary_path = resolved_output_dir / "backend" / "curated.exe"
    manifest_path = manifest_dir / "release.json"
    status = "failed"
    artifact_paths: list[str] = []
    notes = f"versionSource={version_info['source']}; BuildStamp={build_stamp_value}"

    try:
        print("==> Publishing Curated release")
        print(f"Version   : {version_info['version']}")
        print(f"Source    : {version_info['source']}")
        print(f"BuildStamp: {build_stamp_value}")
        print(f"Output    : {resolved_output_dir}")

        build_frontend(version_info["version"], output_dir=str(resolved_output_dir / "frontend"))
        build_backend(
            version_info["version"],
            build_stamp_value,
            output_dir=str(resolved_output_dir / "backend"),
        )
        assemble_release(
            version_info["version"],
            build_stamp_value,
            binary_path=str(binary_path),
            frontend_dist_dir=str(resolved_output_dir / "frontend"),
            output_dir=str(assembled_dir),
        )
        portable_result = package_portable(
            version=version_info["version"],
            input_dir=str(assembled_dir),
            output_dir=str(resolved_output_dir / "portable"),
            history_path=history_path,
            skip_history=True,
        )
        installer_result = package_installer(
            version=version_info["version"],
            app_dir=str(assembled_dir),
            output_dir=str(resolved_output_dir / "installer"),
            history_path=history_path,
            skip_history=True,
        )

        manifest_dir.mkdir(parents=True, exist_ok=True)
        manifest = {
            "productName": "Curated",
            "version": version_info["version"],
            "buildStamp": build_stamp_value,
            "channel": "release",
            "generatedAtUtc": datetime.now(timezone.utc).isoformat(),
            "artifacts": [],
        }
        if portable_zip.exists():
            manifest["artifacts"].append(
                {
                    "type": "portable",
                    "fileName": portable_zip.name,
                    "path": str(portable_zip),
                    "sha256": _sha256_file(portable_zip),
                }
            )
        if installer_exe.exists():
            manifest["artifacts"].append(
                {
                    "type": "installer",
                    "fileName": installer_exe.name,
                    "path": str(installer_exe),
                    "sha256": _sha256_file(installer_exe),
                }
            )
        manifest_path.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")

        artifact_paths.extend(portable_result.get("artifact_paths", []))
        artifact_paths.extend(installer_result.get("artifact_paths", []))
        artifact_paths.append(str(manifest_path))

        result_statuses = [portable_result.get("status"), installer_result.get("status")]
        if "failed" in result_statuses:
            status = "failed"
        elif "partial" in result_statuses:
            status = "partial"
        else:
            status = "success"

        portable_notes = portable_result.get("notes")
        if portable_notes:
            notes = f"{notes}; portable={portable_notes}"
        installer_notes = installer_result.get("notes")
        if installer_notes:
            notes = f"{notes}; installer={installer_notes}"

        print("Release publish flow complete.")
        return {
            "version": version_info["version"],
            "status": status,
            "artifact_paths": artifact_paths,
            "notes": notes,
        }
    except Exception as exc:
        notes = f"{notes}; error={exc}"
        raise
    finally:
        if version_info["version"]:
            history_artifacts = [path for path in artifact_paths if path]
            if not history_artifacts:
                for candidate in (portable_zip, installer_exe, manifest_path):
                    if candidate.exists():
                        history_artifacts.append(str(candidate))
            if history_artifacts:
                _append_history_row(
                    repo_root=repo_root,
                    history_path=history_path,
                    version=version_info["version"],
                    build_type="release:publish",
                    artifact_paths=history_artifacts,
                    status=status,
                    operator=_release_operator(),
                    notes=notes,
                )


def migrate_history(
    markdown_path: str = LEGACY_HISTORY_MD,
    csv_path: str = DEFAULT_HISTORY_CSV,
) -> Path:
    repo_root = get_repo_root()
    resolved_markdown_path = resolve_release_path(markdown_path, repo_root)
    resolved_csv_path = resolve_release_path(csv_path, repo_root)
    migrate_markdown_history_to_csv(resolved_markdown_path, resolved_csv_path)
    return resolved_csv_path


def _resolve_release_version(repo_root: Path, version: str | None, version_file: str) -> dict[str, str]:
    if version and str(version).strip():
        return {"version": str(version).strip(), "source": "explicit"}
    result = allocate_next_patch_in_file(resolve_release_path(version_file, repo_root))
    return {"version": str(result["version"]), "source": "auto-patch"}


def _append_history_row(
    repo_root: Path,
    history_path: str,
    version: str,
    build_type: str,
    artifact_paths: list[str],
    status: str,
    operator: str,
    notes: str,
) -> None:
    resolved_history_path = resolve_release_path(history_path, repo_root)
    _ensure_history_csv(repo_root, resolved_history_path)
    commit = current_short_commit(repo_root)
    branch = current_branch(repo_root)
    previous_commit = _previous_commit(resolved_history_path)
    change_summary = build_package_history_change_summary(
        previous_commit=previous_commit,
        current_commit=commit,
        resolve_commit=lambda value: _safe_resolve_commit(repo_root, value),
        load_git_log=lambda start, end: load_git_log(repo_root, start, end),
    )
    append_history_entry(
        resolved_history_path,
        {
            "date": datetime.now().strftime("%Y-%m-%d"),
            "version": version,
            "commit": commit,
            "branch": branch,
            "build_type": build_type,
            "artifact_paths": "; ".join(
                to_repo_relative_path(path, repo_root) for path in artifact_paths if str(path).strip()
            ),
            "status": _status_label(status),
            "operator": operator,
            "change_summary": change_summary,
            "notes": notes,
        },
    )


def _ensure_history_csv(repo_root: Path, csv_path: Path) -> None:
    if csv_path.exists():
        return
    legacy_markdown_path = resolve_release_path(LEGACY_HISTORY_MD, repo_root)
    if legacy_markdown_path.exists():
        migrate_markdown_history_to_csv(legacy_markdown_path, csv_path)


def _previous_commit(csv_path: Path) -> str | None:
    from .history import extract_previous_history_commit

    return extract_previous_history_commit(csv_path)


def _safe_resolve_commit(repo_root: Path, commit: str) -> str | None:
    try:
        return resolve_commit(repo_root, commit)
    except subprocess.CalledProcessError:
        return None


def _release_operator() -> str:
    return os.environ.get("USERNAME") or os.environ.get("USER") or "unknown"


def _status_label(status: str) -> str:
    return {
        "success": "成功",
        "failed": "失败",
        "partial": "部分成功",
    }.get(status, status)


def _run(command: list[str], cwd: Path, env: dict[str, str] | None = None) -> None:
    resolved_command = list(command)
    executable = shutil.which(resolved_command[0])
    if not executable and os.name == "nt":
        executable = shutil.which(f"{resolved_command[0]}.cmd") or shutil.which(f"{resolved_command[0]}.exe")
    if executable:
        resolved_command[0] = executable
    completed = subprocess.run(resolved_command, cwd=cwd, env=env, check=False)
    if completed.returncode != 0:
        raise RuntimeError(f"{' '.join(command)} failed with exit code {completed.returncode}")


def _zip_directory_contents(input_dir: Path, zip_path: Path) -> None:
    with ZipFile(zip_path, "w", compression=ZIP_DEFLATED) as archive:
        for file_path in sorted(input_dir.rglob("*")):
            if file_path.is_file():
                archive.write(file_path, file_path.relative_to(input_dir))


def _find_iscc() -> Path | None:
    discovered = shutil.which("ISCC.exe")
    if discovered:
        return Path(discovered)
    for candidate in (
        Path(r"C:\Program Files (x86)\Inno Setup 6\ISCC.exe"),
        Path(r"C:\Program Files\Inno Setup 6\ISCC.exe"),
    ):
        if candidate.exists():
            return candidate
    return None


def _sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest().upper()
