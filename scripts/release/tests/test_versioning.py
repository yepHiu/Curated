import json
import shutil
import sys
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[3]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from scripts.release.release_lib.versioning import (  # type: ignore[import-not-found]
    allocate_next_patch_in_file,
    format_version,
    read_version_state,
    set_version_base_in_file,
)


class VersioningTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_root = REPO_ROOT / ".tmp-release-tests" / self.id().replace(".", "_")
        if self.temp_root.exists():
            shutil.rmtree(self.temp_root)
        self.temp_root.mkdir(parents=True, exist_ok=True)

    def tearDown(self) -> None:
        if self.temp_root.exists():
            shutil.rmtree(self.temp_root)

    def write_version_file(self, directory: Path, payload: dict) -> Path:
        target = directory / "version.json"
        target.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
        return target

    def test_read_version_state_and_format(self) -> None:
        file_path = self.write_version_file(
            self.temp_root,
            {
                "schema": 1,
                "current": {"major": 1, "minor": 2, "patch": 6},
            },
        )

        state = read_version_state(file_path)

        self.assertEqual(1, state["schema"])
        self.assertEqual("1.2.6", format_version(state["current"]))

    def test_allocate_next_patch_in_file(self) -> None:
        file_path = self.write_version_file(
            self.temp_root,
            {
                "schema": 1,
                "current": {"major": 1, "minor": 2, "patch": 6},
            },
        )

        result = allocate_next_patch_in_file(file_path)

        self.assertEqual("1.2.7", result["version"])
        saved = json.loads(file_path.read_text(encoding="utf-8"))
        self.assertEqual(7, saved["current"]["patch"])

    def test_set_version_base_resets_patch(self) -> None:
        file_path = self.write_version_file(
            self.temp_root,
            {
                "schema": 1,
                "current": {"major": 9, "minor": 9, "patch": 9},
            },
        )

        result = set_version_base_in_file(file_path, 2, 5)

        self.assertEqual("2.5.0", result["version"])
        saved = json.loads(file_path.read_text(encoding="utf-8"))
        self.assertEqual(
            {"major": 2, "minor": 5, "patch": 0},
            saved["current"],
        )


if __name__ == "__main__":
    unittest.main()
