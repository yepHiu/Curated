from __future__ import annotations

import csv
import re
from pathlib import Path
from typing import Callable


CSV_FIELDNAMES = [
    "date",
    "version",
    "commit",
    "branch",
    "build_type",
    "artifact_paths",
    "status",
    "operator",
    "change_summary",
    "notes",
]

_TABLE_DIVIDER = re.compile(r"^\|\s*-+\s*\|")
_COMMIT_BRANCH = re.compile(r"`?([0-9a-fA-F]{7,40})`?\s*/\s*`?([^`|]+?)`?$")


def read_history_rows(csv_path: str | Path) -> list[dict[str, str]]:
    path = Path(csv_path)
    if not path.exists():
        return []
    with path.open("r", encoding="utf-8-sig", newline="") as handle:
        reader = csv.DictReader(handle)
        return [{key: value or "" for key, value in row.items()} for row in reader]


def append_history_entry(csv_path: str | Path, row: dict[str, str]) -> None:
    path = Path(csv_path)
    path.parent.mkdir(parents=True, exist_ok=True)
    file_exists = path.exists()
    encoding = "utf-8" if file_exists else "utf-8-sig"
    with path.open("a", encoding=encoding, newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=CSV_FIELDNAMES)
        if not file_exists:
            writer.writeheader()
        writer.writerow({field: row.get(field, "") for field in CSV_FIELDNAMES})


def extract_previous_history_commit(csv_path: str | Path) -> str | None:
    rows = read_history_rows(csv_path)
    if not rows:
        return None
    commit = rows[-1].get("commit", "").strip()
    return commit or None


def build_package_history_change_summary(
    previous_commit: str | None,
    current_commit: str,
    resolve_commit: Callable[[str], str | None],
    load_git_log: Callable[[str, str], list[str]],
) -> str:
    if not previous_commit:
        return "首条打包记录，无上一包可比对"

    previous_resolved = resolve_commit(previous_commit)
    if not previous_resolved:
        return "无法解析上一条打包记录对应 commit"

    current_resolved = resolve_commit(current_commit)
    if not current_resolved:
        return "无法解析当前打包记录对应 commit"

    if previous_resolved == current_resolved:
        return "无代码差异（同一提交重复打包）"

    lines = load_git_log(previous_resolved, current_resolved)
    if not lines:
        return "无代码差异（同一提交重复打包）"
    return "\n".join(lines)


def migrate_markdown_history_to_csv(markdown_path: str | Path, csv_path: str | Path) -> None:
    markdown = Path(markdown_path).read_text(encoding="utf-8")
    rows = _parse_markdown_history(markdown)
    target = Path(csv_path)
    target.parent.mkdir(parents=True, exist_ok=True)
    with target.open("w", encoding="utf-8-sig", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=CSV_FIELDNAMES)
        writer.writeheader()
        for row in rows:
            writer.writerow(row)


def _parse_markdown_history(markdown: str) -> list[dict[str, str]]:
    records: list[dict[str, str]] = []
    for raw_line in markdown.splitlines():
        line = raw_line.strip()
        if not line.startswith("|"):
            continue
        if "日期" in line and "版本" in line and "产物路径" in line:
            continue
        if _TABLE_DIVIDER.match(line):
            continue

        cells = [cell.strip() for cell in line.strip("|").split("|")]
        if len(cells) != 9:
            continue

        commit_match = _COMMIT_BRANCH.match(cells[2])
        if not commit_match:
            continue

        build_type = _strip_backticks(cells[3])
        if not build_type:
            continue

        records.append(
            {
                "date": cells[0],
                "version": cells[1],
                "commit": commit_match.group(1).strip(),
                "branch": commit_match.group(2).strip(),
                "build_type": build_type,
                "artifact_paths": _normalize_markdown_cell(cells[4]),
                "status": _normalize_markdown_cell(cells[5]),
                "operator": _normalize_markdown_cell(cells[6]),
                "change_summary": _normalize_markdown_cell(cells[7]).replace("<br>", "\n"),
                "notes": _normalize_markdown_cell(cells[8]),
            }
        )

    return records


def _normalize_markdown_cell(value: str) -> str:
    return value.replace("；", "; ").replace("`", "").strip()


def _strip_backticks(value: str) -> str:
    return value.strip().strip("`").strip()
