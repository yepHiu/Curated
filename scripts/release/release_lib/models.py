from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class VersionState:
    schema: int
    current: dict[str, int]


@dataclass(frozen=True)
class ReleaseResult:
    version: str
    status: str
    artifact_paths: list[str]
    notes: str
