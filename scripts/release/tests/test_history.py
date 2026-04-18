import csv
import shutil
import sys
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[3]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from scripts.release.release_lib.history import (  # type: ignore[import-not-found]
    CSV_FIELDNAMES,
    append_history_entry,
    build_package_history_change_summary,
    extract_previous_history_commit,
    migrate_markdown_history_to_csv,
    read_history_rows,
)


class HistoryTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_root = REPO_ROOT / ".tmp-release-tests" / self.id().replace(".", "_")
        if self.temp_root.exists():
            shutil.rmtree(self.temp_root)
        self.temp_root.mkdir(parents=True, exist_ok=True)

    def tearDown(self) -> None:
        if self.temp_root.exists():
            shutil.rmtree(self.temp_root)

    def test_extract_previous_history_commit_from_csv(self) -> None:
        csv_path = self.temp_root / "history.csv"
        append_history_entry(
            csv_path,
            {
                "date": "2026-04-17",
                "version": "1.2.6",
                "commit": "4319daf7",
                "branch": "master",
                "build_type": "release:publish",
                "artifact_paths": "release/portable/Curated-1.2.6-windows-x64.zip",
                "status": "成功",
                "operator": "wujiahui",
                "change_summary": "no diff",
                "notes": "versionSource=auto-patch",
            },
        )

        self.assertEqual("4319daf7", extract_previous_history_commit(csv_path))

    def test_build_change_summary_uses_git_range(self) -> None:
        summary = build_package_history_change_summary(
            previous_commit="prevsha",
            current_commit="currsha",
            resolve_commit=lambda commit: {
                "prevsha": "prevresolved",
                "currsha": "currresolved",
            }.get(commit),
            load_git_log=lambda start, end: [
                f"{start}..{end}",
                "1111111 feat: add python release cli",
                "2222222 docs: migrate package history to csv",
            ],
        )

        self.assertEqual(
            "prevresolved..currresolved\n"
            "1111111 feat: add python release cli\n"
            "2222222 docs: migrate package history to csv",
            summary,
        )

    def test_migrate_markdown_history_to_csv(self) -> None:
        markdown = """# 整包打包历史

| 日期 | 版本 | 提交 / 分支 | 打包类型 | 产物路径 | 状态 | 操作人 | 变更内容 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 2026-04-17 | 1.2.6 | `4319daf7` / `master` | `release:publish` | `release/portable/Curated-1.2.6-windows-x64.zip`; `release/installer/Curated-Setup-1.2.6.exe` | 成功 | wujiahui | 4319daf7 feat: add python release cli<br>2222222 docs: csv history | BuildStamp=`20260416.165419` |
"""
        markdown_path = self.temp_root / "history.md"
        csv_path = self.temp_root / "history.csv"
        markdown_path.write_text(markdown, encoding="utf-8")

        migrate_markdown_history_to_csv(markdown_path, csv_path)

        rows = read_history_rows(csv_path)
        self.assertEqual(1, len(rows))
        self.assertEqual(CSV_FIELDNAMES, list(rows[0].keys()))
        self.assertEqual("2026-04-17", rows[0]["date"])
        self.assertEqual("4319daf7", rows[0]["commit"])
        self.assertEqual("master", rows[0]["branch"])
        self.assertIn("release/installer/Curated-Setup-1.2.6.exe", rows[0]["artifact_paths"])
        self.assertEqual(
            "4319daf7 feat: add python release cli\n2222222 docs: csv history",
            rows[0]["change_summary"],
        )

        raw = csv_path.read_text(encoding="utf-8-sig")
        parsed_rows = list(csv.DictReader(raw.splitlines()))
        self.assertEqual("1.2.6", parsed_rows[0]["version"])


if __name__ == "__main__":
    unittest.main()
