# Bundled FFmpeg

Place the FFmpeg runtime that should ship with Curated in this directory when
you want to vendor a fixed runtime in the source tree.

Expected layout on Windows:

```text
backend/third_party/ffmpeg/
  bin/
    ffmpeg.exe
    ffprobe.exe
  LICENSES/
    ...
```

Release packaging copies `backend/third_party` into the assembled app directory as `third_party/`.
If `backend/third_party/ffmpeg/bin/ffmpeg.exe` is missing, the release script
tries to copy a real local FFmpeg installation discovered from Scoop or PATH
into the assembled package. Scoop shims are intentionally ignored; the package
must contain the real executable, not the shim launcher.

Packaging fails fast when no redistributable FFmpeg runtime can be found.

At runtime, Curated prefers the bundled executable at:

```text
third_party/ffmpeg/bin/ffmpeg.exe
```

when `player.ffmpegCommand` is left at the default `ffmpeg`.

Notes:

- Include the upstream FFmpeg license and notices that match the binary you distribute.
- Generated release packages add `README-Curated-Bundle.txt` beside the copied runtime to record the source path used for that package.
- If you intentionally want to override the bundled binary, set `player.ffmpegCommand` to a custom command or absolute path in settings.
