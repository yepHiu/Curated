from __future__ import annotations

import argparse
import csv
import json
import subprocess
import sys
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[4]
RELEASE_CLI = REPO_ROOT / "scripts" / "release" / "release_cli.py"
HISTORY_CSV = REPO_ROOT / "docs" / "package-build-history.csv"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Preview Curated packaging version effects")
    parser.add_argument(
        "--mode",
        required=True,
        choices=["publish", "installer", "portable", "preview", "set-base"],
    )
    parser.add_argument("--current-base-version")
    parser.add_argument("--requested-major")
    parser.add_argument("--requested-minor")
    return parser.parse_args()


def load_current_base_version() -> str:
    completed = subprocess.run(
        [sys.executable, str(RELEASE_CLI), "show-version"],
        cwd=REPO_ROOT,
        check=True,
        capture_output=True,
        text=True,
        encoding="utf-8",
    )
    payload = json.loads(completed.stdout)
    return str(payload["version"])


def parse_optional_int(value: str | None) -> int | None:
    if value is None or not str(value).strip():
        return None
    return int(value)


def load_latest_history_row() -> dict[str, str] | None:
    if not HISTORY_CSV.exists():
        return None

    with HISTORY_CSV.open("r", encoding="utf-8-sig", newline="") as handle:
        rows = list(csv.DictReader(handle))

    if not rows:
        return None

    latest = rows[-1]
    return {
        "date": latest.get("date", "") or "",
        "version": latest.get("version", "") or "",
        "buildType": latest.get("build_type", "") or "",
        "status": latest.get("status", "") or "",
        "commit": latest.get("commit", "") or "",
        "changeSummary": latest.get("change_summary", "") or "",
    }


def main() -> None:
    args = parse_args()
    current_base_version = args.current_base_version or load_current_base_version()
    major_value = parse_optional_int(args.requested_major)
    minor_value = parse_optional_int(args.requested_minor)

    current_parts = current_base_version.split(".")
    major = major_value if major_value is not None else int(current_parts[0])
    minor = minor_value if minor_value is not None else int(current_parts[1])
    base_patch = 0 if major_value is not None or minor_value is not None else int(current_parts[2])
    will_bump_patch = args.mode not in {"preview", "set-base"}
    predicted_patch = base_patch + 1 if will_bump_patch else base_patch

    result = {
        "mode": args.mode,
        "currentBaseVersion": current_base_version,
        "baseVersionAfterChange": f"{major}.{minor}.{base_patch}",
        "predictedVersion": f"{major}.{minor}.{predicted_patch}",
        "willBumpPatch": will_bump_patch,
        "historyPath": str(HISTORY_CSV.relative_to(REPO_ROOT)),
        "latestHistory": load_latest_history_row(),
    }
    print(json.dumps(result, ensure_ascii=False))


if __name__ == "__main__":
    main()
