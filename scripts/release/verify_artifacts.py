"""Fail CI on partial packaging, wrong targets, missing files or checksum mismatch."""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path


def verify(root: Path, version: str, platform: str, arch: str, variant: str) -> list[Path]:
    """Validate the complete manifest and each installer before uploading."""
    manifests = list(root.glob("components-*/installer/components-manifest.json"))
    if len(manifests) != 1:
        raise ValueError(f"Expected one build manifest, found {len(manifests)}")
    manifest_path = manifests[0]
    manifest = json.loads(manifest_path.read_text())
    if manifest.get("status") != "built" or manifest.get("version") != version:
        raise ValueError("Build is partial or has the wrong version")
    expected = {"server", "desktop", "full"} if variant in ("all", "full") else {variant}
    artifacts = manifest.get("artifacts", [])
    if len(artifacts) != len(expected) or {item["variant"] for item in artifacts} != expected:
        raise ValueError("Missing or duplicate component installers")
    result = []
    for item in artifacts:
        suffix = "exe" if platform == "windows" else "pkg"
        name = f"Curated-{item['variant'].title()}-Setup-{version}-{platform}-{arch}.{suffix}"
        if item.get("fileName") != name or any(item.get(key) != value for key, value in {"version": version, "platform": platform, "arch": arch}.items()):
            raise ValueError("Unexpected artifact target or filename")
        artifact = manifest_path.parent / name
        if not artifact.is_file() or artifact.stat().st_size == 0:
            raise ValueError(f"Missing or empty installer: {name}")
        if hashlib.sha256(artifact.read_bytes()).hexdigest().lower() != item["sha256"].lower():
            raise ValueError(f"Checksum mismatch: {name}")
        result.append(artifact)
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--platform", choices=["windows", "macos"], required=True)
    parser.add_argument("--arch", choices=["x64", "arm64"], required=True)
    parser.add_argument("--variant", choices=["all", "full", "server", "desktop"], required=True)
    args = parser.parse_args()
    for artifact in verify(**vars(args)):
        print(artifact)
