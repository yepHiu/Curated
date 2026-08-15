import csv
import shutil
import sys
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[3]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from scripts.prd.message_catalog_lint import CSV_FIELDNAMES, lint_message_catalog  # type: ignore[import-not-found]


class MessageCatalogLintTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_root = REPO_ROOT / ".tmp-prd-tests" / self.id().replace(".", "_")
        if self.temp_root.exists():
            shutil.rmtree(self.temp_root)
        self.temp_root.mkdir(parents=True, exist_ok=True)

    def tearDown(self) -> None:
        if self.temp_root.exists():
            shutil.rmtree(self.temp_root)

    def write_csv(self, rows: list[dict[str, str]]) -> Path:
        target = self.temp_root / "message-catalog.csv"
        with target.open("w", encoding="utf-8-sig", newline="") as handle:
            writer = csv.DictWriter(handle, fieldnames=CSV_FIELDNAMES)
            writer.writeheader()
            writer.writerows(rows)
        return target

    def make_row(self, **overrides: str) -> dict[str, str]:
        row = {
            "id": "MSG-0001",
            "title": "Scan in progress",
            "area": "library-scan",
            "trigger": "A library scan is running",
            "level": "now",
            "toast": "no",
            "center": "now",
            "badge": "none",
            "until": "immediate",
            "group": "",
            "cta": "",
            "status": "specified",
            "source_files": "",
            "updated_at": "2026-08-14",
            "notes": "",
        }
        row.update(overrides)
        return row

    def test_accepts_valid_row(self) -> None:
        csv_path = self.write_csv([self.make_row()])
        self.assertEqual([], lint_message_catalog(csv_path))

    def test_rejects_duplicate_ids(self) -> None:
        csv_path = self.write_csv([self.make_row(), self.make_row(title="Duplicate")])
        errors = lint_message_catalog(csv_path)
        self.assertTrue(any("duplicate id" in error.lower() for error in errors))

    def test_rejects_silent_with_badge(self) -> None:
        csv_path = self.write_csv(
            [
                self.make_row(
                    level="silent",
                    center="none",
                    badge="needs-you",
                )
            ]
        )
        errors = lint_message_catalog(csv_path)
        self.assertTrue(any("silent" in error.lower() for error in errors))

    def test_rejects_notify_with_needs_you_center(self) -> None:
        csv_path = self.write_csv(
            [self.make_row(level="notify", center="needs-you", badge="none")]
        )
        errors = lint_message_catalog(csv_path)
        self.assertTrue(any("notify" in error.lower() for error in errors))


if __name__ == "__main__":
    unittest.main()
