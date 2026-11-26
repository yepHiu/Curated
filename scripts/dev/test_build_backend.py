"""开发后端构建入口必须指向现存脚本，并保持 curated-dev 命名。"""
import json
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]


class DevBackendBuildTests(unittest.TestCase):
    def test_package_script_points_at_existing_builder(self):
        package = json.loads((ROOT / "package.json").read_text(encoding="utf-8"))
        command = package["scripts"]["backend:build:dev"]
        self.assertNotIn("build-backend.ps1", command)
        self.assertIn("scripts/dev/build-backend.cjs", command)
        self.assertTrue((ROOT / "scripts/dev/build-backend.cjs").is_file())

    def test_dev_binary_name_avoids_release_executable(self):
        script = (
            "const {devBackendFileName}=require('./scripts/dev/build-backend.cjs');"
            "if (devBackendFileName('win32')!=='curated-dev.exe') process.exit(2);"
            "if (devBackendFileName('linux')!=='curated-dev') process.exit(3);"
            "if (devBackendFileName('darwin')!=='curated-dev') process.exit(4);"
        )
        completed = subprocess.run(["node", "-e", script], cwd=ROOT, check=False, text=True)
        self.assertEqual(completed.returncode, 0)


if __name__ == "__main__":
    unittest.main()
