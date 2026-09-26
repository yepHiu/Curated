from __future__ import annotations
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from scripts.release.release_lib.component_packaging import COMPONENT_IDS, artifact_name, generate_installers, stage_components, verify_payload


class ComponentPackagingTest(unittest.TestCase):
    def setUp(self):
        """Isolate generated payloads from all existing release artifacts."""
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.repo = Path(__file__).resolve().parents[3]

    def fixtures(self):
        """Create minimal component inputs without requiring Windows build tools."""
        binary = self.root / "curated.exe"; binary.write_bytes(b"server")
        web = self.root / "web"; web.mkdir(); (web / "index.html").write_text("Web")
        runtime = self.root / "electron"; runtime.mkdir(); (runtime / "electron.exe").write_bytes(b"desktop")
        (runtime / "resources.pak").write_bytes(b"runtime")
        main = self.root / "main"; main.mkdir()
        for file in ("main.js", "desktop-shell.js", "connections.js", "discovery.js", "updates.js", "settings.js", "preload.cjs", "launcher-preload.cjs", "backend-process.js"):
            (main / file).write_text("fixture")
        (main / "launcher").mkdir(); (main / "launcher/index.html").write_text("Connect")
        return binary, web, runtime, main

    def test_independent_payloads_and_full_bootstrap_share_component_installers(self):
        """Check payload isolation and the Full wrapper's reuse of component identity."""
        binary, web, runtime, main = self.fixtures()
        with patch("scripts.release.release_lib.build_steps._bundle_ffmpeg_runtime", return_value="fixture"):
            components = stage_components(self.repo, self.root / "payloads", "2.0.0", server_binary=binary, frontend=web, electron_runtime=runtime, electron_main=main)
        self.assertFalse((components["desktop"] / "resources/app/curated.exe").exists())
        self.assertFalse((components["desktop"] / "resources/app/electron-dist/backend-process.js").exists())
        self.assertFalse((components["server"] / "resources.pak").exists())
        scripts = generate_installers(self.repo, components, self.root / "installer", "2.0.0")
        self.assertEqual(len(scripts), 3)
        full = scripts[-1].read_text()
        self.assertIn(artifact_name("server", "2.0.0"), full)
        self.assertIn(artifact_name("desktop", "2.0.0"), full)
        self.assertIn("Uninstallable=no", full)
        for app_id in COMPONENT_IDS.values():
            self.assertIn(app_id, full)
        self.assertIn("Inno Setup: App Path", full)
        self.assertNotIn("{localappdata}", full)
        self.assertNotIn("__", full)
        server = scripts[0].read_text()
        self.assertIn("PrivilegesRequired=lowest", server)
        self.assertIn("IsNewer", server)
        self.assertNotIn("__", server)
        self.assertEqual(json.loads((components["desktop"] / "component.json").read_text())["component"], "desktop")

    def test_desktop_build_does_not_require_server_or_ffmpeg(self):
        """Ensure Desktop-only staging has no dependency on Server build tools."""
        _, _, runtime, main = self.fixtures()
        with patch("scripts.release.release_lib.build_steps._bundle_ffmpeg_runtime") as ffmpeg:
            components = stage_components(self.repo, self.root / "payloads", "2.0.0", electron_runtime=runtime, electron_main=main)
        ffmpeg.assert_not_called()
        self.assertEqual(list(components), ["desktop"])
        (components["desktop"] / "leaked.db").write_text("data")
        with self.assertRaises(ValueError): verify_payload("desktop", components["desktop"])

    def test_invalid_versions_and_existing_payloads_are_rejected(self):
        """Protect script interpolation and preserve existing staged artifacts."""
        with self.assertRaises(ValueError): artifact_name("server", '1.0.0"\n[Run]')
        _, _, runtime, main = self.fixtures()
        output = self.root / "payloads"
        stage_components(self.repo, output, "2.0.0", electron_runtime=runtime, electron_main=main)
        with self.assertRaises(FileExistsError): stage_components(self.repo, output, "2.0.0", electron_runtime=runtime, electron_main=main)

if __name__ == "__main__": unittest.main()
