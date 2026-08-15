import argparse
import csv
from pathlib import Path

CSV_FIELDNAMES = [
    "id",
    "title",
    "area",
    "trigger",
    "level",
    "toast",
    "center",
    "badge",
    "until",
    "group",
    "cta",
    "status",
    "source_files",
    "updated_at",
    "notes",
]

ALLOWED_LEVELS = {"silent", "notify", "needs-you", "now"}
ALLOWED_TOAST = {"yes", "no"}
ALLOWED_CENTER = {"none", "recent", "needs-you", "now"}
ALLOWED_BADGE = {"none", "needs-you"}
ALLOWED_UNTIL = {"immediate", "open", "session", "7d", "resolved"}
ALLOWED_STATUSES = {"specified", "implemented", "deprecated"}


def _normalized(value: str | None) -> str:
    return (value or "").strip()


def lint_message_catalog(path: Path) -> list[str]:
    errors: list[str] = []
    seen_ids: set[str] = set()

    with path.open("r", encoding="utf-8-sig", newline="") as handle:
        reader = csv.DictReader(handle)
        missing_columns = [name for name in CSV_FIELDNAMES if name not in (reader.fieldnames or [])]
        if missing_columns:
            return [f"Missing CSV columns: {', '.join(missing_columns)}"]

        for row_number, row in enumerate(reader, start=2):
            msg_id = _normalized(row.get("id"))
            level = _normalized(row.get("level"))
            toast = _normalized(row.get("toast"))
            center = _normalized(row.get("center"))
            badge = _normalized(row.get("badge"))
            until = _normalized(row.get("until"))
            status = _normalized(row.get("status"))
            title = _normalized(row.get("title"))
            trigger = _normalized(row.get("trigger"))

            if not msg_id:
                errors.append(f"Row {row_number}: missing id")
                continue
            if not msg_id.startswith("MSG-"):
                errors.append(f"Row {row_number}: id '{msg_id}' must start with MSG-")
            if msg_id in seen_ids:
                errors.append(f"Row {row_number}: duplicate id '{msg_id}'")
            else:
                seen_ids.add(msg_id)

            if not title:
                errors.append(f"Row {row_number}: missing title")
            if not trigger:
                errors.append(f"Row {row_number}: missing trigger")
            if level not in ALLOWED_LEVELS:
                errors.append(f"Row {row_number}: unknown level '{level}'")
            if toast not in ALLOWED_TOAST:
                errors.append(f"Row {row_number}: unknown toast '{toast}'")
            if center not in ALLOWED_CENTER:
                errors.append(f"Row {row_number}: unknown center '{center}'")
            if badge not in ALLOWED_BADGE:
                errors.append(f"Row {row_number}: unknown badge '{badge}'")
            if until not in ALLOWED_UNTIL:
                errors.append(f"Row {row_number}: unknown until '{until}'")
            if status not in ALLOWED_STATUSES:
                errors.append(f"Row {row_number}: unknown status '{status}'")

            if level == "silent" and (center != "none" or badge != "none"):
                errors.append(f"Row {row_number}: silent rows must use center=none and badge=none")
            if level == "notify" and (center != "recent" or badge != "none"):
                errors.append(f"Row {row_number}: notify rows must use center=recent and badge=none")
            if level == "needs-you" and (center != "needs-you" or badge != "needs-you"):
                errors.append(
                    f"Row {row_number}: needs-you rows must use center=needs-you and badge=needs-you"
                )
            if level == "now" and (center != "now" or badge != "none"):
                errors.append(f"Row {row_number}: now rows must use center=now and badge=none")

    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description="Lint the Curated message-catalog CSV ledger.")
    parser.add_argument("csv_path", type=Path, help="Path to the message catalog CSV file")
    args = parser.parse_args()

    errors = lint_message_catalog(args.csv_path)
    if errors:
        for error in errors:
            print(error)
        return 1

    print(f"OK: {args.csv_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
