# Movie Import Performance Architecture

## Context

Current movie import uses browser `File` objects and multipart upload:

```text
browser File -> XMLHttpRequest/FormData -> Vite proxy in development -> Go net/http multipart reader -> destination temp file -> rename -> scan
```

This path works for local and remote browsers, but it is not equivalent to local filesystem copy. Even on the same machine, bytes still flow through browser upload APIs and HTTP parsing.

Current backend behavior is already streaming-oriented:

- `POST /api/import/movies` uses `r.MultipartReader()` instead of `ParseMultipartForm`, so the server does not need to materialize the full request body before handling files.
- Each accepted file part is copied to `targetPath/<relativePath>.<taskId>.tmp` with a 4 MiB buffer and then atomically renamed to the final destination.
- Task metadata already reports `copiedBytes`, `totalBytes`, file counts, current file name, conflict items, copy failures, and scan handoff.
- `newHTTPServer` intentionally leaves `ReadTimeout` and `WriteTimeout` at zero so large imports are not cut off by long request bodies or delayed responses.

## Direct Answer

Yes, Curated can use a classic chunk upload + resume + final commit architecture in Go. That is the right next performance and reliability improvement for large browser uploads.

Using `fasthttp` is possible, but it should not be the default first step. The current backend, router, tests, middleware, static hosting, and API surface are built around `net/http`. Replacing the whole server would add broad risk while the biggest import bottleneck is not `net/http` dispatch overhead; it is browser-to-backend upload, multipart framing, disk writes, interruption recovery, and development-time proxying.

The pragmatic direction is:

1. Implement resumable raw-byte chunk endpoints on the existing Go `net/http` server.
2. Avoid Vite proxy for large uploads where possible by targeting the backend origin directly in Web API mode.
3. Benchmark current multipart upload, `net/http` chunk upload, and direct-backend upload.
4. Only introduce `fasthttp` if those measurements show HTTP stack overhead is a real bottleneck after chunking.

## Zero-Copy Boundary

"Zero-copy" needs a narrow definition here.

What Curated can optimize:

- Avoid holding complete files in memory.
- Avoid `ParseMultipartForm` temp files.
- Avoid a second full-file merge pass by writing chunks directly into a preallocated staging file at fixed offsets.
- Avoid Vite proxy buffering and extra process hops for large request bodies.
- Reuse buffers and stream request bodies directly into `os.File`.

What Curated should not promise as portable browser upload zero-copy:

- Browser `File` uploads do not expose an absolute source path to the backend, so the backend cannot use a native local file copy path from drag/drop or `<input type=file>`.
- `sendfile` mainly helps file-to-socket responses, not browser socket-to-destination-file uploads.
- Multipart parsing and Go request body readers still involve user-space copies.
- Socket-to-file zero-copy techniques such as `splice` are OS-specific and not a good Windows-first product foundation.

For Curated's upload-only product decision, the practical performance goal is zero extra full-file copies, bounded memory, direct backend upload, and resumability rather than literal kernel-level zero-copy.

## Recommended Architecture

Keep the existing multipart endpoint as the compatibility path:

- Continue using `POST /api/import/movies` for small files, older frontend flows, and simple multi-file imports.
- Keep its task metadata, conflict behavior, default import path validation, scan trigger, and user-facing errors stable.

Add a resumable upload path for large files:

1. `POST /api/import/movies/uploads`
   - Creates an upload session.
   - Request includes file manifests: `relativePath`, `size`, `lastModified`, optional MIME/type hint, and optional client-side fingerprint.
   - Response returns `uploadId`, normalized files, `fileId`s, recommended `chunkSize`, max parallel chunks, and any already completed ranges if a matching session exists.

2. `GET /api/import/movies/uploads/{uploadId}`
   - Returns session status, per-file completed ranges, bytes received, failed chunks, target path, and task id if already committed.
   - Used for resume after page refresh or network interruption.

3. `PUT /api/import/movies/uploads/{uploadId}/files/{fileId}/chunks/{chunkIndex}`
   - Sends a raw binary body, not multipart.
   - Headers carry `Content-Range` or explicit `X-Curated-Offset`, `X-Curated-Chunk-Size`, and optional `X-Curated-Chunk-SHA256`.
   - Backend validates offset, size, file id, session state, and destination containment before writing.

4. `POST /api/import/movies/uploads/{uploadId}/commit`
   - Validates every file has all required byte ranges.
   - Flushes staging files and commits them to the final destination without overwriting existing files.
   - Starts the restricted scan for the target library root.
   - Completes or partial-fails the existing `import.movies` task shape.

5. `DELETE /api/import/movies/uploads/{uploadId}`
   - Aborts the upload and removes staging data.

## Storage Strategy

Prefer fixed-offset staging files over per-chunk files plus final merge.

Recommended write path:

```text
chunk body -> io.CopyBuffer / file.WriteAt(offset) -> .curated-import/<uploadId>/<fileId>.part -> fsync on commit -> no-overwrite finalization
```

Why this is better than classic chunk-file merge:

- It avoids writing every byte twice.
- Commit becomes validation + finalization instead of read-all-chunks + write-final-file.
- It keeps disk usage close to final file size instead of final size plus chunk directory overhead.
- It still supports resume because completed byte ranges are stored in SQLite.

Use per-chunk files only if implementation simplicity is more important than throughput for the first iteration. If per-chunk files are used, commit must stream chunks sequentially into a temp final file and then rename; this is reliable but costs a second full disk write.

Staging should live under the target library root or a sibling directory on the same volume so final rename can remain atomic. A practical layout:

```text
<target-library-root>/.curated-import/<uploadId>/<fileId>.part
<target-library-root>/.curated-import/<uploadId>/manifest.json
```

The staging directory must be hidden from library scanning and ignored by import conflict checks.

## Transfer Cache and Staging Directories

Current multipart import:

- Browser side: the selected `File` remains a browser-managed handle to the user's chosen file. Curated does not create a visible browser-side cache directory.
- Development proxy side: when uploads go through Vite, the proxy is an extra transfer hop. Curated does not own a stable cache directory there, and large-upload performance should not depend on proxy behavior.
- Go backend side: `POST /api/import/movies` streams each accepted multipart file part directly into a destination-side temp file:

```text
<target-library-root>/<safe-relative-path>.<taskId>.tmp
```

- After a successful copy, the backend renames that temp file to:

```text
<target-library-root>/<safe-relative-path>
```

- Unsupported, invalid, or conflicting files are discarded from the request stream and are not saved as import cache files.

Recommended resumable chunk import:

- Store in-progress uploads under a dedicated hidden staging directory inside the target library root:

```text
<target-library-root>/.curated-import/<uploadId>/<fileId>.part
<target-library-root>/.curated-import/<uploadId>/manifest.json
```

- Keep this directory on the same volume as the final movie path so commit can avoid cross-volume copies and can use no-overwrite same-volume finalization.
- Exclude `.curated-import/` from scanner, organizer, metadata scraping, and user-visible library results.
- Persist resumable state in SQLite, but do not store chunk bytes in SQLite.
- Clean up staging data on successful commit, explicit abort, expired sessions, and startup janitor recovery.

Do not use the repository root, `backend/runtime`, Go build cache, or OS temp as the default movie-byte staging location for large imports. Those locations can be on the wrong disk, can be slower, can break atomic rename, and can unexpectedly consume application/runtime storage.

## Persistence

SQLite should persist enough state to resume after process restart:

- `movie_import_upload_sessions`
  - `upload_id`, `task_id`, `target_library_path_id`, `target_root`, `state`, `created_at`, `updated_at`, `expires_at`
- `movie_import_upload_files`
  - `file_id`, `upload_id`, `relative_path`, `safe_relative_path`, `size`, `staging_path`, `final_path`, `state`, `bytes_received`
- `movie_import_upload_chunks`
  - `upload_id`, `file_id`, `chunk_index`, `offset`, `size`, `sha256`, `received_at`

For fixed-offset writes, chunks can be coalesced into ranges in memory for active sessions, but durable rows make recovery and UI status simpler.

## Frontend Strategy

The frontend should keep the current import dialog flow but switch large files to resumable upload:

- Use `File.slice(start, end)` to upload chunks.
- Keep a small upload queue with bounded concurrency, for example 2 to 4 chunks globally.
- Persist resumable client state in IndexedDB or localStorage keyed by `relativePath + size + lastModified + targetLibraryPathId`.
- Show upload speed, ETA, retry count, and per-file status.
- Retry failed chunks with exponential backoff.
- Allow pause/cancel.
- Continue using `use-scan-task-tracker` and `ScanProgressDock` for the backend copy/commit/scan task.

For development mode, large uploads should prefer the backend origin directly. Passing multi-GB uploads through the Vite proxy adds another process hop and can distort benchmarks.

## Speed Optimization Options

The highest-value speed improvements are mostly about removing extra hops, reducing write amplification, and keeping the upload pipeline full without overwhelming disk I/O.

Recommended order:

1. Direct backend upload
   - In Web API mode, send large upload requests to the Go backend origin instead of through Vite proxy.
   - This removes one process hop and avoids proxy-side buffering behavior during development.

2. Raw binary chunk endpoints
   - Upload each chunk as `application/octet-stream` instead of `multipart/form-data`.
   - Keep metadata in URL params, headers, or the upload-session manifest.
   - This avoids repeated multipart boundary parsing for every chunk.

3. Larger chunks
   - Start around 16 MiB or 32 MiB chunks for movie files.
   - Avoid tiny 1 MiB chunks unless the network is unstable; small chunks increase request overhead, SQLite writes, and scheduler churn.
   - Make `chunkSize` server-recommended so benchmarks can tune it without changing frontend code.

4. Bounded parallel chunk upload
   - Use 2 to 4 concurrent chunks globally, not unlimited parallelism.
   - Parallelism keeps the network pipe full, but too much parallelism hurts spinning disks, causes seek contention, and increases memory pressure.
   - Use adaptive concurrency: reduce on repeated failures or slow disk writes, increase cautiously when throughput is stable.

5. Fixed-offset staging writes
   - Write chunks directly into a single staging file using offsets.
   - Avoid per-chunk files plus final merge when performance matters, because merge writes the full movie again.
   - Pre-size the staging file when the session starts to reduce fragmentation and early disk-full surprises.

6. Commit-time flush, not chunk-time flush
   - Do not `fsync` every chunk.
   - Flush at commit, then finalize without overwriting any existing target file.
   - Per-chunk fsync improves crash durability but can severely reduce throughput on Windows disks.

7. Lightweight integrity checks
   - Prefer optional per-chunk hashes over mandatory full-file hash before upload.
   - If hashing is needed, compute it in a Web Worker so the UI thread stays responsive.
   - Avoid blocking the first byte of upload on hashing the whole file.

8. No compression or base64
   - Video files are already compressed; HTTP compression wastes CPU and normally does not reduce bytes.
   - Base64 increases payload size and should not be used for movie import.

9. Exclude staging from scanning until commit
   - Keep `.curated-import/` ignored by library scanning and metadata workflows.
   - Starting scan work while upload is still writing can compete for disk and CPU.

10. Batch UI and task updates
    - Keep frequent in-memory progress updates, but throttle SQLite task snapshots and frontend polling.
    - Over-persisting progress can become visible overhead on slow disks.

11. Use resume to protect effective speed
    - Resume does not make one clean transfer faster, but it prevents losing multi-GB progress after interruption.
    - Effective completion time improves dramatically on unstable Wi-Fi or remote clients.

Lower-priority options:

- Evaluate `fasthttp` only after the raw chunk path exists and benchmark data shows the HTTP stack is the bottleneck.
- HTTP/2 or HTTP/3 can help some remote networks, but they add deployment complexity and are not the first local Windows desktop optimization.
- Memory-mapped file writes are not recommended for the first version; `os.File.WriteAt` or `io.CopyBuffer` is easier to control and test.

## fasthttp Position

`fasthttp` can be evaluated in one of two scoped ways:

1. Dedicated upload listener
   - Keep all existing APIs on `net/http`.
   - Run a separate `fasthttp` listener only for chunk upload bodies.
   - Higher operational complexity: second listener, CORS/origin handling, shutdown path, config, tests.

2. Dedicated upload sub-server behind the same backend process
   - Keep the application service layer and storage logic shared.
   - Avoid trying to adapt every existing handler to `fasthttp`.
   - Still increases dependency and test surface.

Do not migrate the full Curated backend to `fasthttp` just for imports unless benchmark evidence justifies it. The immediate wins are raw chunk endpoints, offset writes, direct backend upload, and resume semantics.

## MVP Plan

1. Keep the existing multipart endpoint as compatibility behavior.
2. Add upload session DTOs, task metadata fields, and error codes for resumable imports.
3. Add backend session create/status/chunk/commit/abort handlers on `net/http`.
4. Store session and chunk state in SQLite.
5. Write chunks into same-volume staging files by offset.
6. Extend `MovieImportDialog` to use chunk upload above a size threshold.
7. Add upload speed, ETA, retry, pause, and cancel UI.
8. Benchmark:
   - current multipart upload
   - `net/http` chunk upload through Vite proxy
   - `net/http` chunk upload direct to backend
   - optional `fasthttp` upload listener only if needed

## Success Criteria

- A multi-GB remote upload can resume after network interruption, browser refresh, or backend restart.
- Failed chunks retry without restarting the whole file.
- Final commit never overwrites an existing library file.
- Staging files are cleaned up after abort, expiry, or successful commit.
- UI shows speed and ETA.
- Existing default import path behavior, task progress, conflict handling, scan trigger, and notifications continue to work.
- Peak backend memory stays bounded by active chunk concurrency and buffer size, not by file size.

## Implementation Status - 2026-05-02

Decision update:

- Backend local-path copy was removed by product decision.
- Curated keeps upload import as the user-facing movie import model.
- Current implementation uses `POST /api/import/movies` with streamed multipart reads and destination temp files.
- `MovieImportDialog` currently uploads selected files with `XMLHttpRequest` progress.

Still pending:

- SQLite persistence for resumable upload session state after backend restart.
- Startup/expiry janitor for orphaned `.curated-import` staging directories.
- Speed/ETA display.
- Direct-backend upload benchmarking.
- Optional `fasthttp` benchmark after the `net/http` chunk path exists.

Implemented in first optimization slice:

- Backend `net/http` resumable upload endpoints under `/api/import/movies/uploads`.
- Raw binary chunk upload with `X-Curated-Offset` and optional `X-Curated-Chunk-Size`.
- Same-volume `.curated-import/<uploadId>/<fileId>.part` staging under the target library root.
- Commit-time flush/no-overwrite finalization and restricted scan trigger.
- Frontend large-file branch in `api.importMovies`, using the existing `MovieImportDialog` progress shape.
