"""Brand and validate the Apple Silicon Desktop runtime."""
from __future__ import annotations

import plistlib
import shutil
import subprocess
from pathlib import Path


def output(*args: str) -> str:
    """Read a native build tool result, failing on command errors."""
    return subprocess.check_output(args, text=True).strip()


def require_arch(binary: Path, arch: str) -> None:
    """Reject a runtime from a different CPU architecture."""
    if arch != "arm64":
        raise ValueError("macOS supports Apple Silicon Desktop only")
    expected = "arm64"
    if expected not in output("lipo", "-archs", str(binary)).split():
        raise ValueError(f"Wrong architecture for {binary}: expected {expected}")


def brand_desktop(bundle: Path, version: str, arch: str, icon: Path) -> None:
    """Keep Electron's framework structure intact while setting Curated identity."""
    executable = bundle / "Contents/MacOS/Electron"
    require_arch(executable, arch)
    plist = bundle / "Contents/Info.plist"
    with plist.open("rb") as stream:
        info = plistlib.load(stream)
    info.update(CFBundleName="Curated", CFBundleDisplayName="Curated", CFBundleIdentifier="app.curated.desktop",
                CFBundleShortVersionString=version, CFBundleVersion=version, CFBundleIconFile="curated.icns",
                LSMinimumSystemVersion="15.0")
    with plist.open("wb") as stream:
        plistlib.dump(info, stream)
    shutil.copy2(icon, bundle / "Contents/Resources/curated.icns")
    # Ad-hoc signing makes the mutated bundle internally consistent; not notarization.
    subprocess.run(["codesign", "--force", "--deep", "--sign", "-", str(bundle)], check=True, capture_output=True)
    subprocess.run(["codesign", "--verify", "--deep", "--strict", str(bundle)], check=True)
