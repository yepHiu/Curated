import argparse
import csv
from pathlib import Path


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

ALLOWED_STATUSES = {
    "idea",
    "triaged",
    "specified",
    "planned",
    "in_progress",
    "implemented",
    "verified",
    "released",
    "blocked",
    "deferred",
    "rejected",
    "superseded",
}

STATUSES_REQUIRING_ACCEPTANCE = {
    "specified",
    "planned",
    "in_progress",
    "implemented",
    "verified",
    "released",
    "blocked",
}


def _normalized(value: str | None) -> str:
    return (value or "").strip()


def lint_csv(path: Path) -> list[str]:
    errors: list[str] = []
    seen_ids: set[str] = set()

    with path.open("r", encoding="utf-8-sig", newline="") as handle:
        reader = csv.DictReader(handle)
        missing_columns = [name for name in CSV_FIELDNAMES if name not in (reader.fieldnames or [])]
        if missing_columns:
            return [f"Missing CSV columns: {', '.join(missing_columns)}"]

        for row_number, row in enumerate(reader, start=2):
            req_id = _normalized(row.get("id"))
            status = _normalized(row.get("status"))
            acceptance = _normalized(row.get("acceptance_criteria"))
            implementation_refs = _normalized(row.get("implementation_refs"))
            test_refs = _normalized(row.get("test_refs"))

            if req_id in seen_ids:
                errors.append(f"Row {row_number}: duplicate id '{req_id}'")
            elif req_id:
                seen_ids.add(req_id)

            if status not in ALLOWED_STATUSES:
                errors.append(f"Row {row_number}: unknown status '{status}'")

            if status in STATUSES_REQUIRING_ACCEPTANCE and not acceptance:
                errors.append(
                    f"Row {row_number}: acceptance_criteria is required for status '{status}'"
                )

            if status == "implemented" and not implementation_refs:
                errors.append(
                    f"Row {row_number}: implementation_refs is required for status 'implemented'"
                )

            if status == "verified":
                if not implementation_refs:
                    errors.append(
                        f"Row {row_number}: implementation_refs is required for status 'verified'"
                    )
                if not test_refs:
                    errors.append(
                        f"Row {row_number}: test_refs is required for status 'verified'"
                    )

    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description="Lint the repository PRD CSV ledger.")
    parser.add_argument("csv_path", type=Path, help="Path to the PRD CSV file")
    args = parser.parse_args()

    errors = lint_csv(args.csv_path)
    if errors:
        for error in errors:
            print(error)
        return 1

    print(f"OK: {args.csv_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
