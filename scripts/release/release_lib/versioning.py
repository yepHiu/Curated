from __future__ import annotations

import json
from pathlib import Path
from typing import Any


SCHEMA_VERSION = 1


def _assert_whole_number(value: Any, label: str) -> int:
    if not isinstance(value, int) or value < 0:
        raise ValueError(f"Invalid {label}: {value}")
    return value


def normalize_version_state(payload: dict[str, Any]) -> dict[str, Any]:
    if not isinstance(payload, dict):
        raise ValueError("Release version file must be a JSON object.")

    schema = payload.get("schema")
    if schema != SCHEMA_VERSION:
        raise ValueError(
            f"Unsupported release version schema: {schema}. Expected {SCHEMA_VERSION}.",
        )

    current = payload.get("current")
    if not isinstance(current, dict):
        raise ValueError("Release version file is missing a valid current version.")

    major = _assert_whole_number(current.get("major"), "major version")
    minor = _assert_whole_number(current.get("minor"), "minor version")
    patch = _assert_whole_number(current.get("patch"), "patch version")

    return {
        "schema": SCHEMA_VERSION,
        "current": {
            "major": major,
            "minor": minor,
            "patch": patch,
        },
    }


def format_version(version: dict[str, int]) -> str:
    return f"{version['major']}.{version['minor']}.{version['patch']}"


def read_version_state(file_path: str | Path) -> dict[str, Any]:
    path = Path(file_path)
    return normalize_version_state(json.loads(path.read_text(encoding="utf-8")))


def write_version_state(file_path: str | Path, state: dict[str, Any]) -> dict[str, Any]:
    normalized = normalize_version_state(state)
    path = Path(file_path)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(normalized, indent=2) + "\n", encoding="utf-8")
    return normalized


def allocate_next_patch_in_file(file_path: str | Path) -> dict[str, Any]:
    current = read_version_state(file_path)
    next_state = {
        "schema": SCHEMA_VERSION,
        "current": {
            "major": current["current"]["major"],
            "minor": current["current"]["minor"],
            "patch": current["current"]["patch"] + 1,
        },
    }
    saved = write_version_state(file_path, next_state)
    return {"state": saved, "version": format_version(saved["current"])}


def set_version_base_in_file(file_path: str | Path, major: int, minor: int) -> dict[str, Any]:
    next_state = {
        "schema": SCHEMA_VERSION,
        "current": {
            "major": _assert_whole_number(major, "major version"),
            "minor": _assert_whole_number(minor, "minor version"),
            "patch": 0,
        },
    }
    saved = write_version_state(file_path, next_state)
    return {"state": saved, "version": format_version(saved["current"])}
