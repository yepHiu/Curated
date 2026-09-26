"""Relocate Homebrew FFmpeg's Mach-O dependency closure into the Server payload."""
from __future__ import annotations

import hashlib
import json
import os
import plistlib
import shutil
import subprocess
from pathlib import Path


def output(*args: str) -> str:
    """Read a native build tool result, failing on command errors."""
    return subprocess.check_output(args, text=True).strip()


def require_arch(binary: Path, arch: str) -> None:
    """Reject a runtime from a different CPU architecture."""
    expected = "arm64" if arch == "arm64" else "x86_64"
    if expected not in output("lipo", "-archs", str(binary)).split():
        raise ValueError(f"Wrong architecture for {binary}: expected {expected}")


def dependencies(binary: Path) -> list[str]:
    """Read Mach-O load commands without executing the target."""
    return [line.strip().split(" (compatibility version", 1)[0]
            for line in output("otool", "-L", str(binary)).splitlines()[1:]]


def resolve_dependency(name: str, source: Path, executable: Path) -> Path:
    """Resolve loader-relative and rpath links; unresolved links must fail packaging."""
    def expand(value: str) -> Path:
        """Expand dyld path tokens against the original library and executable."""
        return Path(value.replace("@loader_path", str(source.parent)).replace("@executable_path", str(executable.parent)))
    if name.startswith("@rpath/"):
        lines = output("otool", "-l", str(source)).splitlines()
        for index, line in enumerate(lines):
            if line.strip() == "cmd LC_RPATH":
                candidate = expand(lines[index + 2].strip().split("path ", 1)[1].split(" (offset", 1)[0]) / name[7:]
                if candidate.is_file():
                    return candidate.resolve()
        raise FileNotFoundError(f"Unresolved dependency {name} from {source}")
    candidate = expand(name)
    if not candidate.is_absolute() or not candidate.is_file():
        raise FileNotFoundError(f"Unresolved dependency {name} from {source}")
    return candidate.resolve()


def bundle_ffmpeg(destination: Path, arch: str) -> None:
    """Copy binaries/dylibs, rewrite every non-system link, sign and smoke-test."""
    bin_dir = destination / "bin"
    lib_dir = destination / "lib"
    notices = destination / "licenses"
    for directory in (bin_dir, lib_dir, notices):
        directory.mkdir(parents=True, exist_ok=True)
    copied: dict[Path, Path] = {}
    origins: list[dict[str, str]] = []

    def copy(source: Path, target: Path, executable: Path) -> Path:
        """Recursively copy a dependency once and rewrite its references."""
        source = source.resolve()
        if source in copied:
            return copied[source]
        require_arch(source, arch)
        copied[source] = target
        shutil.copy2(source, target)
        target.chmod(0o755)
        # Capture installed formula licensing and provenance alongside the payload.
        formula_root = next((p for p in source.parents if p.parent.parent.name == "Cellar"), None)
        if formula_root:
            notice_dir = notices / f"{formula_root.parent.name}-{formula_root.name}"
            notice_dir.mkdir(exist_ok=True)
            for item in formula_root.iterdir():
                if item.is_file() and (item.name.upper().startswith(("LICENSE", "COPYING", "NOTICE", "AUTHORS")) or item.name == "INSTALL_RECEIPT.json"):
                    shutil.copy2(item, notice_dir / item.name)
        origins.append({"source": str(source), "file": str(target.relative_to(destination)),
                        "sourceSHA256": hashlib.sha256(source.read_bytes()).hexdigest()})
        # Preserve LINKEDIT layout while rewriting; replace invalidated signatures afterwards.
        if target.suffix == ".dylib":
            subprocess.run(["install_name_tool", "-id", f"@rpath/{target.name}", str(target)], check=True)
        for name in dependencies(source):
            if name.startswith(("/usr/lib/", "/System/Library/")):
                continue
            dependency = resolve_dependency(name, source, executable)
            if dependency == source:  # dylib's own LC_ID_DYLIB
                continue
            filename = f"{hashlib.sha256(str(dependency).encode()).hexdigest()[:12]}-{dependency.name}"
            linked = copy(dependency, lib_dir / filename, executable)
            relative = os.path.relpath(linked, target.parent)
            subprocess.run(["install_name_tool", "-change", name, f"@loader_path/{relative}", str(target)], check=True)
        subprocess.run(["codesign", "--force", "--sign", "-", str(target)], check=True, capture_output=True)
        return target

    for name in ("ffmpeg", "ffprobe"):
        source = shutil.which(name)
        if not source:
            raise FileNotFoundError(f"Install native {name} before macOS packaging")
        copy(Path(source), bin_dir / name, Path(source).resolve())
    # Reject any residual build-machine link, even if that machine can resolve it.
    for target in copied.values():
        for name in dependencies(target):
            if not name.startswith(("/usr/lib/", "/System/Library/", "@loader_path/", "@rpath/")):
                raise ValueError(f"Nonportable dependency: {target}: {name}")
    (destination / "runtime-provenance.json").write_text(json.dumps(origins, indent=2) + "\n")
    for name in ("ffmpeg", "ffprobe"):
        subprocess.run([str(bin_dir / name), "-version"], check=True, stdout=subprocess.DEVNULL)


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
