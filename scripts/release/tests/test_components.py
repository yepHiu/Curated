import unittest
import json
from pathlib import Path

from scripts.release.release_lib.components import artifact_name, component_plan


class ComponentTests(unittest.TestCase):
    def test_names(self):
        self.assertEqual(artifact_name("desktop", "0.1.0", "macos", "arm64", "dmg"), "Curated-Desktop-0.1.0-macos-arm64.dmg")
        self.assertEqual(artifact_name("full", "1.6.0", "windows", "x64", "exe"), "Curated-Full-Setup-1.6.0-windows-x64.exe")
        self.assertEqual(artifact_name("server", "1.6.0", "linux", "arm64", "tar.gz"), "Curated-Server-1.6.0-linux-arm64.tar.gz")

    def test_rejects_mislabelled_artifacts(self):
        for values in [("desktop", "0.1.0-beta", "macos", "arm64", "dmg"),
                       ("desktop", "0.1.0", "windows", "universal", "exe"),
                       ("server", "1.6.0", "macos", "arm64", "dmg"),
                       ("desktop", "0.1.0", "windows", "x64", "dmg")]:
            with self.subTest(values=values), self.assertRaises(ValueError):
                artifact_name(*values)

    def test_full_keeps_exact_component_versions_without_allocating(self):
        root = Path(__file__).resolve().parents[3]
        sources = list((root / "scripts/release/versions").glob("*.json"))
        sources.append(root / "backend/internal/version/server.json")
        before = {p: p.read_bytes() for p in sources}
        plan = component_plan(root, "full", "windows", "x64", "exe")
        self.assertEqual(plan["status"], "planned")
        for component, source in (("desktop", root / "scripts/release/versions/desktop.json"), ("server", root / "backend/internal/version/server.json")):
            state = json.loads(before[source])["current"]
            self.assertEqual(plan["components"][component], ".".join(str(state[key]) for key in ("major", "minor", "patch")))
        self.assertEqual(before, {p: p.read_bytes() for p in sources})
