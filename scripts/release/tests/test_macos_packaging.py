"""Installer contracts can be checked on every OS without invoking Apple tools."""
import hashlib
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch
from xml.etree import ElementTree as ET

from scripts.release.release_lib.macos_packaging import artifact_name, write_distribution, publish_macos
from scripts.release.release_lib.build_steps import build_backend
from scripts.release.verify_artifacts import verify


class MacOSPackagingTest(unittest.TestCase):
    def setUp(self):
        """Keep packaging fixtures outside existing build artifacts."""
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)

    def test_distribution_selects_both_stable_receipts_and_native_architecture(self):
        """Both Full components are selected and CPU/OS requirements are explicit."""
        for arch, native in (("arm64", "arm64"), ("x64", "x86_64")):
            path = self.root / f"{arch}.xml"
            write_distribution(path, ["server", "desktop"], "1.2.3", arch)
            root = ET.parse(path).getroot()
            self.assertEqual(root.find("options").attrib["hostArchitectures"], native)
            self.assertEqual(root.find("volume-check/allowed-os-versions/os-version").attrib["min"], "15.0")
            self.assertEqual([item.attrib["id"] for item in root.findall("pkg-ref")], ["app.curated.server", "app.curated.desktop"])
            self.assertTrue(all(item.attrib["start_selected"] == "true" for item in root.findall("choice")))

    def test_backend_darwin_build_never_uses_windows_linker_flag(self):
        """Native Server builds must have matching Go OS/architecture flags."""
        with patch("scripts.release.release_lib.build_steps.get_repo_root", return_value=self.root), patch("scripts.release.release_lib.build_steps._run") as run:
            build_backend("1.2.3", "test", binary_name="curated", target_os="darwin", target_arch="arm64")
        self.assertEqual(run.call_args.kwargs["env"]["GOOS"], "darwin")
        self.assertEqual(run.call_args.kwargs["env"]["GOARCH"], "arm64")
        self.assertNotIn("windowsgui", " ".join(run.call_args.args[0]))

    def test_wrong_host_fails_before_allocating_version(self):
        """Cross-CPU execution must fail without consuming a release version."""
        with patch("platform.system", return_value="Windows"), patch("scripts.release.release_lib.build_steps._resolve_release_version") as version:
            with self.assertRaises(ValueError):
                publish_macos(version="1.2.3")
        version.assert_not_called()
        with self.assertRaises(ValueError):
            artifact_name("desktop", "1.2.3;bad", "arm64")

    def test_verify_rejects_partial_missing_wrong_target_and_tampered_artifacts(self):
        """A green build requires the requested complete, untampered installers."""
        folder = self.root / "components-test/installer"
        folder.mkdir(parents=True)
        name = artifact_name("desktop", "1.2.3", "arm64")
        artifact = folder / name
        artifact.write_bytes(b"installer")
        manifest = {"version": "1.2.3", "status": "built", "artifacts": [{"variant": "desktop", "version": "1.2.3", "platform": "macos", "arch": "arm64", "fileName": name, "sha256": hashlib.sha256(b"installer").hexdigest()}]}
        manifest_path = folder / "components-manifest.json"
        manifest_path.write_text(json.dumps(manifest))
        self.assertEqual(verify(self.root, "1.2.3", "macos", "arm64", "desktop"), [artifact])
        for variant, arch in (("all", "arm64"), ("desktop", "x64")):
            with self.assertRaises(ValueError): verify(self.root, "1.2.3", "macos", arch, variant)
        manifest["status"] = "scripts-only"
        manifest_path.write_text(json.dumps(manifest))
        with self.assertRaises(ValueError): verify(self.root, "1.2.3", "macos", "arm64", "desktop")
        manifest["status"] = "built"
        manifest_path.write_text(json.dumps(manifest))
        artifact.write_bytes(b"tampered")
        with self.assertRaises(ValueError): verify(self.root, "1.2.3", "macos", "arm64", "desktop")
        artifact.unlink()
        with self.assertRaises(ValueError): verify(self.root, "1.2.3", "macos", "arm64", "desktop")
