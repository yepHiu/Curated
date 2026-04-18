from __future__ import annotations

import subprocess
from pathlib import Path


def run_git(repo_root: Path, *args: str) -> str:
    completed = subprocess.run(
        ["git", *args],
        cwd=repo_root,
        check=True,
        capture_output=True,
        text=True,
        encoding="utf-8",
    )
    return completed.stdout.strip()


def resolve_commit(repo_root: Path, commit: str) -> str:
    return run_git(repo_root, "rev-parse", "--verify", commit)


def current_short_commit(repo_root: Path) -> str:
    return run_git(repo_root, "rev-parse", "--short=8", "HEAD")


def current_branch(repo_root: Path) -> str:
    branch = run_git(repo_root, "branch", "--show-current")
    if branch:
        return branch
    return run_git(repo_root, "rev-parse", "--abbrev-ref", "HEAD")


def load_git_log(repo_root: Path, start_commit: str, end_commit: str) -> list[str]:
    output = run_git(repo_root, "log", "--oneline", f"{start_commit}..{end_commit}")
    return [line.strip() for line in output.splitlines() if line.strip()]
