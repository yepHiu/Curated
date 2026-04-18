from __future__ import annotations

from pathlib import Path


def get_repo_root() -> Path:
    return Path(__file__).resolve().parents[3]


def resolve_release_path(path_value: str | Path, repo_root: Path | None = None) -> Path:
    candidate = Path(path_value)
    if candidate.is_absolute():
        return candidate.resolve()
    root = repo_root or get_repo_root()
    return (root / candidate).resolve()


def to_repo_relative_path(path_value: str | Path, repo_root: Path | None = None) -> str:
    root = (repo_root or get_repo_root()).resolve()
    target = Path(path_value).resolve()
    return target.relative_to(root).as_posix()
