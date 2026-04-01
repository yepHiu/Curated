# Bundled FFmpeg

Place the FFmpeg runtime that should ship with Curated in this directory.

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

At runtime, Curated prefers the bundled executable at:

```text
third_party/ffmpeg/bin/ffmpeg.exe
```

when `player.ffmpegCommand` is left at the default `ffmpeg`.

Notes:

- Include the upstream FFmpeg license and notices that match the binary you distribute.
- If you intentionally want to override the bundled binary, set `player.ffmpegCommand` to a custom command or absolute path in settings.
