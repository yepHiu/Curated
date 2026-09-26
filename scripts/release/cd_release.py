"""Validation and GitHub publication for Windows and Mac Desktop CD.

Builds remain in release_cli.py. This helper never allocates versions or moves tags.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import urllib.error
import urllib.request

import sys
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
from scripts.release.release_lib.macos_desktop import desktop_version, mac_artifact_names


TAG_PATTERN = re.compile(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)")


def version_from_tag(tag: str) -> str:
    if not TAG_PATTERN.fullmatch(tag):
        raise ValueError("Release tag must be vMAJOR.MINOR.PATCH (no suffix or leading zeros)")
    return tag[1:]


def git(root: Path, *args: str) -> str:
    return subprocess.check_output(["git", *args], cwd=root, text=True).strip()


def source_commit(root: Path, tag: str) -> str:
    version_from_tag(tag)
    commit = git(root, "rev-parse", "--verify", f"refs/tags/{tag}^{{commit}}")
    if git(root, "rev-parse", "HEAD") != commit:
        raise ValueError("Checkout does not match the release tag commit")
    return commit


def release_body(root: Path, version: str) -> str:
    notes = list((root / "docs/release-notes").glob(f"*-release-{version}-notes.md"))
    if len(notes) != 1:
        raise ValueError(f"Expected exactly one release notes file for {version}; found {len(notes)}")
    text = notes[0].read_text(encoding="utf-8")
    sections = re.split(r"(?m)^## GitHub Release Body\s*$", text)
    if len(sections) != 2:
        raise ValueError("Release notes require exactly one ## GitHub Release Body heading")
    body = re.split(r"(?m)^## ", sections[1], maxsplit=1)[0].strip()
    if not re.sub(r"<!--.*?-->", "", body, flags=re.S).strip():
        raise ValueError("GitHub Release Body must not be empty")
    return body + "\n"


def prepare(root: Path, tag: str, output: Path) -> dict:
    version = version_from_tag(tag)
    commit = source_commit(root, tag)
    state = json.loads((root / "scripts/release/version.json").read_text(encoding="utf-8"))
    if state.get("schema") != 1 or state.get("current") != dict(zip(("major", "minor", "patch"), map(int, version.split(".")))):
        raise ValueError("Tag and scripts/release/version.json must agree before release")
    body = release_body(root, version)
    metadata = {"tag": tag, "version": version, "commit": commit}
    output.mkdir(parents=True, exist_ok=True)
    (output / "metadata.json").write_text(json.dumps(metadata, indent=2) + "\n", encoding="utf-8")
    (output / "release-body.md").write_text(body, encoding="utf-8")
    return metadata


def sha256(path: Path) -> str:
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def artifact_names(version: str) -> list[str]:
    return [f"Curated-{version}-windows-x64.zip", f"Curated-Setup-{version}.exe", "release.json"]


def stage(root: Path, tag: str, output: Path) -> None:
    metadata = prepare(root, tag, output)
    version = metadata["version"]
    manifest = json.loads((root / "release/manifest/release.json").read_text(encoding="utf-8"))
    if manifest.get("version") != version or manifest.get("channel") != "release":
        raise ValueError("Manifest version/channel does not match this release")
    expected = dict(zip(("portable", "installer"), artifact_names(version)[:2]))
    entries = manifest.get("artifacts", [])
    if len(entries) != 2 or {a.get("type") for a in entries} != set(expected):
        raise ValueError("Both installer and portable artifacts are required; partial builds cannot publish")
    # Validate everything before copying files or writing the public manifest.
    for entry in entries:
        name = expected[entry["type"]]
        path = root / "release" / entry["type"] / name
        if entry.get("fileName") != name or not path.is_file() or path.stat().st_size == 0:
            raise ValueError(f"Missing or invalid release artifact: {name}")
        if sha256(path) != str(entry.get("sha256", "")).lower():
            raise ValueError(f"Artifact checksum mismatch: {name}")
    assets = output / "assets"
    assets.mkdir(exist_ok=True)
    for entry in entries:
        name = expected[entry["type"]]
        shutil.copy2(root / "release" / entry["type"] / name, assets / name)
        entry["path"] = name  # Do not publish runner-local filesystem paths.
    manifest["sourceCommit"] = metadata["commit"]
    (assets / "release.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    checksums = "".join(f"{sha256(assets / name)}  {name}\n" for name in artifact_names(version))
    (assets / "SHA256SUMS.txt").write_text(checksums, encoding="utf-8")


def verify_staged(output: Path, metadata: dict) -> None:
    assets = output / "assets"
    names = artifact_names(metadata["version"])
    if "desktopVersion" in metadata:
        names += mac_artifact_names(metadata["desktopVersion"]) + ["desktop-macos.json"]
    if {p.name for p in assets.iterdir()} != {*names, "SHA256SUMS.txt"}:
        raise ValueError("Unexpected or missing staged release assets")
    expected = "".join(f"{sha256(assets / name)}  {name}\n" for name in names)
    if (assets / "SHA256SUMS.txt").read_text(encoding="utf-8") != expected:
        raise ValueError("Downloaded release assets failed checksum verification")
    manifest = json.loads((assets / "release.json").read_text(encoding="utf-8"))
    if manifest.get("version") != metadata["version"] or manifest.get("sourceCommit") != metadata["commit"]:
        raise ValueError("Downloaded artifacts do not belong to the selected source commit")


def merge_macos(root: Path, output: Path, macos: Path, metadata: dict) -> dict:
    verify_staged(output, metadata)
    version = desktop_version(root)
    names = mac_artifact_names(version)
    manifest = json.loads((macos / "desktop-macos.json").read_text(encoding="utf-8"))
    required = {"schema": 1, "component": "desktop", "version": version, "platform": "macos",
                "arch": "arm64", "distribution": "desktop", "sourceCommit": metadata["commit"]}
    if any(manifest.get(key) != value for key, value in required.items()):
        raise ValueError("Mac Desktop manifest does not match the release source/version/architecture")
    entries = manifest.get("artifacts", [])
    if len(entries) != 2 or {entry.get("fileName") for entry in entries} != set(names):
        raise ValueError("Mac Desktop requires both DMG and ZIP")
    for entry in entries:
        path = macos / entry["fileName"]
        if not path.is_file() or path.stat().st_size == 0 or sha256(path) != entry.get("sha256"):
            raise ValueError("Mac Desktop artifact checksum mismatch")
    assets = output / "assets"
    for name in [*names, "desktop-macos.json"]:
        if (assets / name).exists():
            raise FileExistsError(assets / name)
        shutil.copy2(macos / name, assets / name)
    all_names = artifact_names(metadata["version"]) + names + ["desktop-macos.json"]
    (assets / "SHA256SUMS.txt").write_text("".join(f"{sha256(assets / name)}  {name}\n" for name in all_names), encoding="utf-8")
    metadata = {**metadata, "desktopVersion": version}
    (output / "metadata.json").write_text(json.dumps(metadata, indent=2) + "\n", encoding="utf-8")
    return metadata


def api(path: str, payload: dict | None = None, method: str | None = None):
    request = urllib.request.Request(
        f"https://api.github.com/repos/{os.environ['GITHUB_REPOSITORY']}/{path}",
        data=None if payload is None else json.dumps(payload).encode(),
        method=method,
        headers={"Authorization": f"Bearer {os.environ['GH_TOKEN']}",
                 "Accept": "application/vnd.github+json", "X-GitHub-Api-Version": "2022-11-28",
                 "Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
            return json.load(response)
    except urllib.error.HTTPError as error:
        if error.code == 404 and path.startswith("releases/tags/") and payload is None:
            return None
        detail = error.read(8192).decode('utf-8', errors='replace') if error.fp is not None else ''
        error.add_note(f'GitHub API {method or ("POST" if payload is not None else "GET")} {path}: {detail}')
        raise


def source_marker(metadata: dict) -> str:
    return f"<!-- curated-cd-source:{metadata['commit']} -->"


def find_release(tag: str) -> dict | None:
    release = api(f"releases/tags/{tag}")
    if release is not None:
        return release
    # The by-tag REST endpoint only finds published releases. Authenticated
    # listing includes drafts; paginate so older unfinished drafts are found.
    page = 1
    while True:
        releases = api(f"releases?per_page=100&page={page}")
        matches = [item for item in releases if item.get("tag_name") == tag]
        if matches:
            if len(matches) != 1:
                raise ValueError("Multiple releases use the selected tag")
            return matches[0]
        if len(releases) < 100:
            return None
        page += 1


def check_release(metadata: dict) -> dict | None:
    remote = api(f"git/ref/tags/{metadata['tag']}")["object"]
    for _ in range(8):
        if remote["type"] != "tag":
            break
        remote = api(f"git/tags/{remote['sha']}")["object"]
    if remote["type"] != "commit" or remote["sha"] != metadata["commit"]:
        raise ValueError("Remote tag no longer points to the validated release commit")
    release = find_release(metadata["tag"])
    if release is not None:
        if not release.get("draft"):
            raise ValueError("This version is already published; refusing to overwrite it")
        # GitHub may ignore target_commitish when the tag already exists.
        # Record draft ownership independently of that advisory API field.
        if source_marker(metadata) not in release.get("body", ""):
            raise ValueError("Existing draft belongs to a different or unpinned source commit")
    return release


def verify_uploaded(release: dict, assets: Path) -> None:
    uploaded = {entry["name"]: entry for entry in release["assets"]}
    expected = {p.name for p in assets.iterdir()}
    if set(uploaded) != expected:
        raise ValueError("GitHub Release contains missing or unexpected assets")
    for path in assets.iterdir():
        entry = uploaded[path.name]
        if entry.get("size") != path.stat().st_size or entry.get("digest") != f"sha256:{sha256(path)}":
            raise ValueError(f"GitHub upload verification failed: {path.name}")


def publish(root: Path, tag: str, output: Path, mode: str, macos: Path | None = None) -> None:
    metadata = json.loads((output / "metadata.json").read_text(encoding="utf-8"))
    if metadata != {"tag": tag, "version": version_from_tag(tag), "commit": source_commit(root, tag)}:
        raise ValueError("Release metadata does not match the checked-out tag")
    if macos is not None:
        metadata = merge_macos(root, output, macos, metadata)
    verify_staged(output, metadata)
    body = release_body(root, metadata["version"]) + "\n" + source_marker(metadata) + "\n"
    if macos is not None:
        body += "\n### macOS Desktop (Apple Silicon)\n\n"
        body += f"Standalone Desktop {metadata['desktopVersion']} connects to an existing Curated Server. "
        body += "Requires Apple Silicon; Intel Macs are not supported. Ad-hoc signed, not Apple notarized.\n\n"
        body += "\n".join(f"- `{name}`" for name in mac_artifact_names(metadata['desktopVersion'])) + "\n"
    release = check_release(metadata)
    payload = {"tag_name": tag, "target_commitish": metadata["commit"],
               "name": f"Curated {tag}", "body": body, "draft": True, "prerelease": False}
    if release is None:
        release = api("releases", payload)
    else:
        release = api(f"releases/{release['id']}", payload, "PATCH")
    # gh streams large binaries. --clobber only applies to the verified unpublished draft.
    subprocess.run(["gh", "release", "upload", tag, "--repo", os.environ["GITHUB_REPOSITORY"],
                    "--clobber", *map(str, sorted((output / "assets").iterdir()))], check=True)
    release = api(f"releases/{release['id']}")
    verify_uploaded(release, output / "assets")
    if mode == "publish":
        check_release(metadata)
        release = api(f"releases/{release['id']}", {"draft": False, "make_latest": "true"}, "PATCH")
    print(f"Release {'published' if mode == 'publish' else 'draft ready'}: {release['html_url']}")
    if summary := os.environ.get("GITHUB_STEP_SUMMARY"):
        with open(summary, "a", encoding="utf-8") as stream:
            stream.write(f"## Curated {tag}\n\n[{mode}]({release['html_url']}) · `{metadata['commit']}`\n")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("prepare", "stage", "check", "publish"))
    parser.add_argument("--tag", required=True)
    parser.add_argument("--output", type=Path, default=Path(".workspace/cd-release"))
    parser.add_argument("--mode", choices=("draft", "publish"), default="draft")
    parser.add_argument("--macos-dir", type=Path, help="Required Mac Desktop artifacts for combined CD publication")
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[2]
    if args.command == "stage":
        stage(root, args.tag, args.output)
    elif args.command == "publish":
        publish(root, args.tag, args.output, args.mode, args.macos_dir)
    else:
        metadata = prepare(root, args.tag, args.output)
        if args.command == "check":
            check_release(metadata)
        if target := os.environ.get("GITHUB_OUTPUT"):
            with open(target, "a", encoding="utf-8") as stream:
                stream.writelines(f"{key}={value}\n" for key, value in metadata.items())


if __name__ == "__main__":
    main()
