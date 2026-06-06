import os
import shutil
import sys
import unittest
from pathlib import Path
from unittest.mock import patch


REPO_ROOT = Path(__file__).resolve().parents[3]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

import scripts.release.release_lib.build_steps as build_steps  # type: ignore[import-not-found]
from scripts.release.release_lib.build_steps import assemble_release, build_backend  # type: ignore[import-not-found]


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

    def test_build_backend_uses_explicit_release_go_dirs_from_environment(self) -> None:
        captured: dict[str, object] = {}
        go_cache_dir = self.temp_root / ".workspace" / "release-go-cache"
        go_tmp_dir = self.temp_root / ".workspace" / "release-go-tmp"

        def fake_run(command: list[str], cwd: Path, env: dict[str, str] | None = None) -> None:
            captured["command"] = command
            captured["cwd"] = cwd
            captured["env"] = dict(env or {})

        with patch.dict(
            os.environ,
            {
                "CURATED_RELEASE_GOCACHE": str(go_cache_dir),
                "CURATED_RELEASE_GOTMPDIR": str(go_tmp_dir),
            },
        ):
            with patch("scripts.release.release_lib.build_steps.get_repo_root", return_value=self.temp_root):
                with patch("scripts.release.release_lib.build_steps._run", side_effect=fake_run):
                    build_backend("1.2.3", "20260424.010203", output_dir="release/backend")

        env = captured["env"]
        self.assertIsInstance(env, dict)
        self.assertEqual(Path(str(env["GOCACHE"])), go_cache_dir.resolve())
        self.assertEqual(Path(str(env["GOTMPDIR"])), go_tmp_dir.resolve())
        self.assertTrue(go_cache_dir.is_dir())
        self.assertTrue(go_tmp_dir.is_dir())

    def test_bundle_ffmpeg_runtime_copies_discovered_real_binaries(self) -> None:
        source_bin_dir = self.temp_root / "tools" / "ffmpeg" / "bin"
        source_bin_dir.mkdir(parents=True)
        ffmpeg_source = source_bin_dir / "ffmpeg.exe"
        ffprobe_source = source_bin_dir / "ffprobe.exe"
        ffmpeg_source.write_bytes(b"real ffmpeg")
        ffprobe_source.write_bytes(b"real ffprobe")
        release_dir = self.temp_root / "release" / "Curated"
        release_dir.mkdir(parents=True)

        runtime = build_steps.FFmpegRuntime(
            ffmpeg=ffmpeg_source,
            ffprobe=ffprobe_source,
            source="test-runtime",
        )

        with patch(
            "scripts.release.release_lib.build_steps._discover_ffmpeg_runtime",
            return_value=runtime,
        ):
            source = build_steps._bundle_ffmpeg_runtime(self.temp_root, release_dir)

        self.assertEqual(source, "test-runtime")
        self.assertEqual(
            (release_dir / "third_party" / "ffmpeg" / "bin" / "ffmpeg.exe").read_bytes(),
            b"real ffmpeg",
        )
        self.assertEqual(
            (release_dir / "third_party" / "ffmpeg" / "bin" / "ffprobe.exe").read_bytes(),
            b"real ffprobe",
        )
        notice = release_dir / "third_party" / "ffmpeg" / "README-Curated-Bundle.txt"
        self.assertIn("test-runtime", notice.read_text(encoding="utf-8"))

    def test_bundle_ffmpeg_runtime_prefers_repository_runtime(self) -> None:
        repo_bin_dir = self.temp_root / "backend" / "third_party" / "ffmpeg" / "bin"
        repo_bin_dir.mkdir(parents=True)
        (repo_bin_dir / "ffmpeg.exe").write_bytes(b"repo ffmpeg")
        (repo_bin_dir / "ffprobe.exe").write_bytes(b"repo ffprobe")
        release_dir = self.temp_root / "release" / "Curated"
        release_dir.mkdir(parents=True)

        source = build_steps._bundle_ffmpeg_runtime(self.temp_root, release_dir)

        self.assertEqual(source, "backend/third_party")
        self.assertEqual(
            (release_dir / "third_party" / "ffmpeg" / "bin" / "ffmpeg.exe").read_bytes(),
            b"repo ffmpeg",
        )
        self.assertEqual(
            (release_dir / "third_party" / "ffmpeg" / "bin" / "ffprobe.exe").read_bytes(),
            b"repo ffprobe",
        )

    def test_bundle_ffmpeg_runtime_fails_when_no_runtime_is_available(self) -> None:
        release_dir = self.temp_root / "release" / "Curated"
        release_dir.mkdir(parents=True)

        with patch(
            "scripts.release.release_lib.build_steps._discover_ffmpeg_runtime",
            return_value=None,
        ):
            with self.assertRaises(FileNotFoundError):
                build_steps._bundle_ffmpeg_runtime(self.temp_root, release_dir)

    def test_discover_ffmpeg_runtime_rejects_scoop_shims_and_uses_scoop_real_path(self) -> None:
        source_bin_dir = self.temp_root / "scoop" / "apps" / "ffmpeg" / "current" / "bin"
        source_bin_dir.mkdir(parents=True)
        ffmpeg_source = source_bin_dir / "ffmpeg.exe"
        ffprobe_source = source_bin_dir / "ffprobe.exe"
        ffmpeg_source.write_bytes(b"real ffmpeg")
        ffprobe_source.write_bytes(b"real ffprobe")
        shim_path = self.temp_root / "scoop" / "shims" / "ffmpeg.exe"
        shim_path.parent.mkdir(parents=True)
        shim_path.write_bytes(b"shim")

        def fake_run_capture(command: list[str]) -> str | None:
            if command == ["scoop", "which", "ffmpeg"]:
                return str(ffmpeg_source)
            if command == ["scoop", "which", "ffprobe"]:
                return str(ffprobe_source)
            return None

        with patch("scripts.release.release_lib.build_steps.shutil.which", return_value=str(shim_path)):
            runtime = build_steps._discover_ffmpeg_runtime(
                self.temp_root,
                run_capture=fake_run_capture,
            )

        self.assertIsNotNone(runtime)
        self.assertEqual(runtime.ffmpeg, ffmpeg_source)
        self.assertEqual(runtime.ffprobe, ffprobe_source)
        self.assertEqual(runtime.source, "scoop")
        self.assertNotEqual(runtime.ffmpeg, shim_path)

    def test_assemble_release_stages_electron_desktop_app(self) -> None:
        backend_binary = self.temp_root / "release" / "backend" / "curated.exe"
        backend_binary.parent.mkdir(parents=True)
        backend_binary.write_bytes(b"go backend")

        frontend_dist = self.temp_root / "release" / "frontend"
        frontend_dist.mkdir(parents=True)
        (frontend_dist / "index.html").write_text("<!doctype html>", encoding="utf-8")

        electron_dist = self.temp_root / "node_modules" / "electron" / "dist"
        electron_dist.mkdir(parents=True)
        (electron_dist / "electron.exe").write_bytes(b"electron runtime")
        (electron_dist / "resources.pak").write_bytes(b"pak")
        (electron_dist / "resources").mkdir()

        compiled_main = self.temp_root / "electron-dist"
        compiled_main.mkdir()
        (compiled_main / "main.js").write_text("console.log('main')", encoding="utf-8")
        (compiled_main / "preload.cjs").write_text("", encoding="utf-8")

        asset_dir = self.temp_root / "backend" / "internal" / "assets"
        asset_dir.mkdir(parents=True)
        (asset_dir / "curated.ico").write_bytes(b"ico")

        config_dir = self.temp_root / "config"
        config_dir.mkdir()
        (config_dir / "library-config.cfg").write_text("{}", encoding="utf-8")

        plan_dir = self.temp_root / "docs" / "plan"
        plan_dir.mkdir(parents=True)
        (plan_dir / "2026-03-31-production-packaging-and-config-strategy.md").write_text(
            "packaging",
            encoding="utf-8",
        )

        repo_ffmpeg_dir = self.temp_root / "backend" / "third_party" / "ffmpeg" / "bin"
        repo_ffmpeg_dir.mkdir(parents=True)
        (repo_ffmpeg_dir / "ffmpeg.exe").write_bytes(b"ffmpeg")
        (repo_ffmpeg_dir / "ffprobe.exe").write_bytes(b"ffprobe")

        with patch("scripts.release.release_lib.build_steps.get_repo_root", return_value=self.temp_root):
            output = assemble_release(
                "1.4.7",
                "20260514.010203",
                binary_path="release/backend/curated.exe",
                frontend_dist_dir="release/frontend",
                electron_main_dir="electron-dist",
                electron_runtime_dir="node_modules/electron/dist",
                output_dir="release/Curated",
            )

        app_dir = output / "resources" / "app"
        self.assertEqual((output / "Curated.exe").read_bytes(), b"electron runtime")
        self.assertFalse((output / "electron.exe").exists())
        self.assertEqual((app_dir / "curated.exe").read_bytes(), b"go backend")
        self.assertTrue((app_dir / "frontend-dist" / "index.html").is_file())
        self.assertTrue((app_dir / "electron-dist" / "main.js").is_file())
        self.assertTrue((app_dir / "electron-dist" / "preload.cjs").is_file())
        self.assertTrue((app_dir / "third_party" / "ffmpeg" / "bin" / "ffmpeg.exe").is_file())
        self.assertEqual((app_dir / "package.json").read_text(encoding="utf-8"), '{\n  "name": "curated-desktop",\n  "version": "1.4.7",\n  "type": "module",\n  "main": "electron-dist/main.js"\n}\n')
        release_notes = (output / "README-release.txt").read_text(encoding="utf-8")
        self.assertIn("Curated.exe is the Electron desktop shell", release_notes)
        self.assertIn("resources\\app\\curated.exe is the release Go backend", release_notes)

    def test_installer_template_launches_electron_desktop_exe(self) -> None:
        template = (REPO_ROOT / "scripts" / "release" / "windows" / "Curated.iss.tpl").read_text(encoding="utf-8")

        self.assertIn('#define MyAppExeName "Curated.exe"', template)
        self.assertIn('Filename: "{app}\\{#MyAppExeName}"', template)
        self.assertIn("CloseApplicationsFilter=Curated.exe", template)

    def test_publish_release_builds_electron_main_before_assembly(self) -> None:
        calls: list[str] = []

        def record(name: str):
            def inner(*args: object, **kwargs: object) -> object:
                calls.append(name)
                if name == "package-portable":
                    return {"status": "success", "artifact_paths": [], "notes": "portable"}
                if name == "package-installer":
                    return {"status": "partial", "artifact_paths": [], "notes": "installer"}
                return None

            return inner

        with patch("scripts.release.release_lib.build_steps.get_repo_root", return_value=self.temp_root):
            with patch("scripts.release.release_lib.build_steps._resolve_release_version", return_value={"version": "1.4.7", "source": "explicit"}):
                with patch("scripts.release.release_lib.build_steps.build_frontend", side_effect=record("build-frontend")):
                    with patch("scripts.release.release_lib.build_steps.build_backend", side_effect=record("build-backend")):
                        with patch("scripts.release.release_lib.build_steps.build_electron_main", side_effect=record("build-electron-main")):
                            with patch("scripts.release.release_lib.build_steps.assemble_release", side_effect=record("assemble-release")):
                                with patch("scripts.release.release_lib.build_steps.package_portable", side_effect=record("package-portable")):
                                    with patch("scripts.release.release_lib.build_steps.package_installer", side_effect=record("package-installer")):
                                        build_steps.publish_release(
                                            version="1.4.7",
                                            build_stamp="20260514.010203",
                                            output_dir=str(self.temp_root / "release"),
                                            history_path=str(self.temp_root / "history.csv"),
                                        )

        self.assertEqual(
            calls[:4],
            ["build-frontend", "build-backend", "build-electron-main", "assemble-release"],
        )


if __name__ == "__main__":
    unittest.main()
