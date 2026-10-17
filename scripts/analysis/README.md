# Curated Code Atlas

Internal repository explorer. Run commands from the repository root:

```powershell
pnpm code:atlas
```

Open `http://127.0.0.1:8795`. Requires installed project dependencies, Git, and Go. The tool starts its own Node service; it does not use the product server or database. `--port 8796` selects another loopback port.

- **Refresh:** a Worker scans the current working directory, including uncommitted files. One scan runs at a time. Failed refreshes preserve the previous successful result; scans never write a report automatically.
- **Export:** download the current successful snapshot from the UI, or run `pnpm code:atlas:export` to regenerate `docs/reference/code-atlas.html`.
- **Build:** `pnpm code:atlas:build` writes `.workspace/code-atlas-dist/`; `pnpm code:atlas:preview` serves that build with the same live APIs.
- **Check:** `pnpm code:atlas:check` runs Vue type checking and Node tests. No additional dependencies or package manager lockfiles.

## Structure

| File | Responsibility |
| --- | --- |
| `code-atlas.mjs` / `go-atlas.go` | Repository inventory, AST analysis, dependency resolution, offline rendering |
| `scan-worker.mjs` / `scan-store.mjs` | Worker lifecycle, progress, single active job, last successful result |
| `server.mjs` | Local HTTP API, Vite development middleware or built UI serving |
| `app/App.vue` | Refresh/export controls, loading/errors, status synchronization |
| `app/AtlasExplorer.vue` | Vue lifecycle adapter for the shared explorer in Shadow DOM |
| `app/explorer.*` | Shared map, functions, dependency graph, file inspector; offline export uses the same code |

The four `/api/atlas/` endpoints are documented in `docs/guide.md`. Snapshot revision includes a process session ID so clients refresh correctly after a server restart. No persistent scan cache or automatic file watching is implemented. Function and branch coverage is explained within the explorer; this is static analysis, not a runtime call graph.
