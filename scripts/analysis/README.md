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

## Exploring the dependency network

Select **依赖脉络**, then **展开画布** for a viewport-sized workspace. Every file in the selected scope is retained, including disconnected files and direct local dependency targets outside the scope. Choose **全部仓库文件** to include tests, documentation, configuration and assets; these may have no parsed import connections.

Drag to pan; use the wheel or a two-finger pinch to zoom around the pointer. The minimap jumps to another region, **适应全图** fits the network, and Escape exits the expanded view. A focused canvas also supports arrow keys, `+` / `-`, and `0` to fit. Search highlights matches without removing the rest of the graph; click a result (or press Enter) to locate a node. Click a node to inspect incoming/outgoing imports and open its file details. Zoom and selection survive view switches and rescans; scope changes and **重置视图** reset them.

Circles represent files; diamonds represent Go packages. Solid arrows are static local imports, while dashed lines show Go package membership, not function calls. Directory clusters provide stable spatial orientation. File names appear as you zoom in. The list loads results in batches, but the Canvas graph has no Top N truncation. The offline HTML includes the same network and controls.

## Structure

| File | Responsibility |
| --- | --- |
| `code-atlas.mjs` / `go-atlas.go` | Repository inventory, AST analysis, dependency resolution, offline rendering |
| `scan-worker.mjs` / `scan-store.mjs` | Worker lifecycle, progress, single active job, last successful result |
| `server.mjs` | Local HTTP API, Vite development middleware or built UI serving |
| `app/App.vue` | Refresh/export controls, loading/errors, status synchronization |
| `app/AtlasExplorer.vue` | Vue lifecycle adapter for the shared explorer in Shadow DOM |
| `app/explorer.*` | Shared map, functions, dependency graph, file inspector; offline export uses the same code |
| `app/network-model.mjs` / `app/network-canvas.mjs` | Complete file/package graph, deterministic directory layout, Canvas rendering and navigation |

The four `/api/atlas/` endpoints are documented in `docs/guide.md`. Snapshot revision includes a process session ID so clients refresh correctly after a server restart. No persistent scan cache or automatic file watching is implemented. Function and branch coverage is explained within the explorer; this is static analysis, not a runtime call graph.
