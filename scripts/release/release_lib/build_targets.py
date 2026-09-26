"""Supported installer matrix shared by Actions and release-contract tests."""


def selected_targets(platform: str, variant: str) -> list[dict[str, str]]:
    """Select Windows components and Apple Silicon Desktop, rejecting empty choices."""
    if platform not in ("all", "windows", "macos") or variant not in ("all", "full", "server", "desktop"):
        raise ValueError("Unsupported build selection")
    targets = []
    if platform in ("all", "windows"):
        targets.append({"os": "windows-2022", "platform": "windows", "arch": "x64", "variant": variant})
    if platform in ("all", "macos") and variant in ("all", "desktop"):
        targets.append({"os": "macos-15", "platform": "macos", "arch": "arm64", "variant": "desktop"})
    if not targets:
        raise ValueError("macOS supports Apple Silicon Desktop only; choose all or desktop")
    return targets
