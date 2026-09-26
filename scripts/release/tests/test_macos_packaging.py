"""Installer contracts can be checked on every OS without invoking Apple tools."""
import hashlib
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch
from xml.etree import ElementTree as ET

from scripts.release.release_lib.macos_packaging import artifact_name, write_distribution, publish_macos
from scripts.release.release_lib.build_targets import selected_targets
from scripts.release.verify_artifacts import verify


class MacOSPackagingTest(unittest.TestCase):
    def setUp(self):
        """Keep packaging fixtures outside existing build artifacts."""
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)

    def test_distribution_only_selects_apple_silicon_desktop(self):
        """The installer contains one Desktop receipt and no Server payload."""
        path = self.root / "desktop.xml"
        write_distribution(path, ["desktop"], "1.2.3", "arm64")
        root = ET.parse(path).getroot()
        self.assertEqual(root.find("options").attrib["hostArchitectures"], "arm64")
        self.assertEqual([item.attrib["id"] for item in root.findall("pkg-ref")], ["app.curated.desktop"])
        for components, arch in ((["server"], "arm64"), (["desktop"], "x64")):
            with self.assertRaises(ValueError): write_distribution(path, components, "1.2.3", arch)

    def test_matrix_respects_platform_and_component_scope(self):
        """Default is four packages; Server/Full selections never run on a Mac."""
        targets = selected_targets("all", "all")
        self.assertEqual([(t["platform"], t["arch"], t["variant"]) for t in targets], [("windows", "x64", "all"), ("macos", "arm64", "desktop")])
        for variant in ("server", "full"):
            self.assertEqual([t["platform"] for t in selected_targets("all", variant)], ["windows"])
            with self.assertRaises(ValueError): selected_targets("macos", variant)
        self.assertEqual(len(selected_targets("all", "desktop")), 2)
        self.assertEqual(selected_targets("macos", "all")[0]["variant"], "desktop")

    def test_unsupported_packages_fail_before_allocating_version(self):
        """CLI rejects macOS Server, Full and Intel before touching any artifacts."""
        with patch("scripts.release.release_lib.build_steps._resolve_release_version") as version:
            for variant, arch in (("server", "arm64"), ("full", "arm64"), ("desktop", "x64")):
                with self.assertRaises(ValueError): publish_macos(variant=variant, arch=arch)
                with self.assertRaises(ValueError): artifact_name(variant, "1.2.3", arch)
        version.assert_not_called()

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
        self.assertEqual(verify(self.root, "1.2.3", "macos", "arm64", "all"), [artifact])
        for variant, arch in (("server", "arm64"), ("full", "arm64"), ("desktop", "x64")):
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
