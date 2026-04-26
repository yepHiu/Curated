import shutil
import sys
import unittest
from pathlib import Path
from unittest.mock import patch


REPO_ROOT = Path(__file__).resolve().parents[3]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

import scripts.release.release_lib.build_steps as build_steps  # type: ignore[import-not-found]
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


if __name__ == "__main__":
    unittest.main()
