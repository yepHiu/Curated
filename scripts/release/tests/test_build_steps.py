import shutil
import sys
import unittest
from pathlib import Path
from unittest.mock import patch


REPO_ROOT = Path(__file__).resolve().parents[3]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from scripts.release.release_lib.build_steps import build_backend  # type: ignore[import-not-found]


class BuildStepsTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_root = REPO_ROOT / ".tmp-release-tests" / self.id().replace(".", "_")
        if self.temp_root.exists():
            shutil.rmtree(self.temp_root)
        (self.temp_root / "backend").mkdir(parents=True, exist_ok=True)

    def tearDown(self) -> None:
        if self.temp_root.exists():
            shutil.rmtree(self.temp_root)

    def test_build_backend_uses_non_repo_go_cache_dirs(self) -> None:
        captured: dict[str, object] = {}

        def fake_run(command: list[str], cwd: Path, env: dict[str, str] | None = None) -> None:
            captured["command"] = command
            captured["cwd"] = cwd
            captured["env"] = dict(env or {})

        with patch("scripts.release.release_lib.build_steps.get_repo_root", return_value=self.temp_root):
            with patch("scripts.release.release_lib.build_steps._run", side_effect=fake_run):
                build_backend("1.2.3", "20260424.010203", output_dir="release/backend")

        env = captured["env"]
        self.assertIsInstance(env, dict)
        self.assertIn("GOCACHE", env)
        self.assertIn("GOTMPDIR", env)

        repo_root_text = str(self.temp_root.resolve())
        self.assertFalse(str(env["GOCACHE"]).startswith(repo_root_text))
        self.assertFalse(str(env["GOTMPDIR"]).startswith(repo_root_text))


if __name__ == "__main__":
    unittest.main()
