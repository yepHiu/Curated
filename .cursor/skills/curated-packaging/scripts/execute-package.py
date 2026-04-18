from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[4]
RELEASE_CLI = REPO_ROOT / "scripts" / "release" / "release_cli.py"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Execute Curated packaging workflows")
    parser.add_argument(
        "--mode",
        required=True,
        choices=["publish", "installer", "portable", "set-base"],
    )
    parser.add_argument("--major", type=int)
    parser.add_argument("--minor", type=int)
    return parser.parse_args()


def build_command(args: argparse.Namespace) -> list[str]:
    if args.mode == "publish":
        return [sys.executable, str(RELEASE_CLI), "publish"]
    if args.mode == "installer":
        return [sys.executable, str(RELEASE_CLI), "package-installer"]
    if args.mode == "portable":
        return [sys.executable, str(RELEASE_CLI), "package-portable"]
    if args.major is None or args.minor is None:
        raise ValueError("Mode set-base requires --major and --minor.")
    return [
        sys.executable,
        str(RELEASE_CLI),
        "set-version-base",
        "--major",
        str(args.major),
        "--minor",
        str(args.minor),
    ]


def main() -> None:
    args = parse_args()
    command = build_command(args)
    completed = subprocess.run(command, cwd=REPO_ROOT, check=False)
    if completed.returncode != 0:
        raise SystemExit(completed.returncode)


if __name__ == "__main__":
    main()
