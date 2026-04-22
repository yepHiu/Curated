import csv
import shutil
import sys
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[3]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from scripts.prd.prd_lint import lint_csv  # type: ignore[import-not-found]


CSV_FIELDNAMES = [
    "id",
    "title",
    "type",
    "area",
    "priority",
    "status",
    "progress",
    "source",
    "problem",
    "proposal",
    "acceptance_criteria",
    "dependencies",
    "owner",
    "target_version",
    "implementation_refs",
    "test_refs",
    "detail_doc",
    "updated_at",
    "notes",
]


class PrdLintTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_root = REPO_ROOT / ".tmp-prd-tests" / self.id().replace(".", "_")
        if self.temp_root.exists():
            shutil.rmtree(self.temp_root)
        self.temp_root.mkdir(parents=True, exist_ok=True)

    def tearDown(self) -> None:
        if self.temp_root.exists():
            shutil.rmtree(self.temp_root)

    def write_csv(self, rows: list[dict[str, str]]) -> Path:
        target = self.temp_root / "requirements.csv"
        with target.open("w", encoding="utf-8-sig", newline="") as handle:
            writer = csv.DictWriter(handle, fieldnames=CSV_FIELDNAMES)
            writer.writeheader()
            writer.writerows(rows)
        return target

    def make_row(self, **overrides: str) -> dict[str, str]:
        row = {
            "id": "REQ-0001",
            "title": "Bootstrap repo-local PRD CSV workflow",
            "type": "ops",
            "area": "product-process",
            "priority": "P1",
            "status": "specified",
            "progress": "0",
            "source": "user-idea",
            "problem": "Loose requirements are hard to track and refine.",
            "proposal": "Use a git-tracked CSV ledger plus a repo-local skill.",
            "acceptance_criteria": "CSV exists; README documents workflow; lint passes",
            "dependencies": "",
            "owner": "wujiahui",
            "target_version": "prd-bootstrap",
            "implementation_refs": "",
            "test_refs": "",
            "detail_doc": "",
            "updated_at": "2026-04-22",
            "notes": "",
        }
        row.update(overrides)
        return row

    def test_accepts_valid_seeded_csv(self) -> None:
        csv_path = self.write_csv([self.make_row()])

        self.assertEqual([], lint_csv(csv_path))

    def test_rejects_duplicate_ids(self) -> None:
        csv_path = self.write_csv(
            [
                self.make_row(id="REQ-0001"),
                self.make_row(id="REQ-0001", title="Duplicate ID row"),
            ]
        )

        errors = lint_csv(csv_path)

        self.assertTrue(any("duplicate id" in error.lower() for error in errors))

    def test_rejects_unknown_status(self) -> None:
        csv_path = self.write_csv([self.make_row(status="ship-it")])

        errors = lint_csv(csv_path)

        self.assertTrue(any("unknown status" in error.lower() for error in errors))

    def test_rejects_implemented_without_implementation_refs(self) -> None:
        csv_path = self.write_csv([self.make_row(status="implemented")])

        errors = lint_csv(csv_path)

        self.assertTrue(
            any("implementation_refs" in error.lower() for error in errors)
        )

    def test_rejects_verified_without_test_refs(self) -> None:
        csv_path = self.write_csv(
            [
                self.make_row(
                    status="verified",
                    implementation_refs="docs/plan/2026-04-22-prd-csv-bootstrap-implementation-plan.md",
                )
            ]
        )

        errors = lint_csv(csv_path)

        self.assertTrue(any("test_refs" in error.lower() for error in errors))

    def test_rejects_specified_without_acceptance_criteria(self) -> None:
        csv_path = self.write_csv([self.make_row(acceptance_criteria="")])

        errors = lint_csv(csv_path)

        self.assertTrue(
            any("acceptance_criteria" in error.lower() for error in errors)
        )


if __name__ == "__main__":
    unittest.main()
