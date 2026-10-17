import { test } from "node:test";
import assert from "node:assert/strict";
import { createScanStore, runWorker } from "./scan-store.mjs";
import { startAtlasServer } from "./server.mjs";
import { renderOfflineHtml } from "./code-atlas.mjs";
import vm from "node:vm";
import http from "node:http";

const snapshot = () => ({
  schema: 1,
  generatedAt: new Date().toISOString(),
  commit: "fixture",
  branch: "test",
  dirty: false,
  files: [],
  functions: [],
  edges: [],
  warnings: [],
  unresolved: [],
  excluded: [],
  counts: { files: 0, source: 0, sourceLines: 0, functions: 0, parsedFiles: 0 },
});

test("atlas and product dev servers resolve separate dependency caches", async () => {
  const { resolveConfig } = await import("vite");
  const { fileURLToPath } = await import("node:url");
  const config = async (relative) =>
    resolveConfig({
      configFile: fileURLToPath(new URL(relative, import.meta.url)),
      configLoader: "native",
    }, "serve");
  const product = await config("../../vite.config.ts");
  const atlas = await config("./vite.config.ts");
  assert.notEqual(atlas.cacheDir, product.cacheDir);
});

test("concurrent scans share a task; failed refresh preserves successful data and can retry", async () => {
  let resolve,
    reject,
    progress,
    runs = 0;
  const store = createScanStore((onProgress) => {
    runs++;
    progress = onProgress;
    return new Promise((yes, no) => {
      resolve = yes;
      reject = no;
    });
  });
  const first = store.scan();
  assert.equal(store.scan(), first);
  await Promise.resolve();
  assert.equal(runs, 1);
  progress({ phase: "go", message: "go fixture" });
  assert.equal(store.status().phase, "go");
  const result = snapshot();
  resolve(result);
  await first;
  assert.equal(store.snapshot(), result);
  assert.equal(store.status().revision, 1);
  const failed = store.scan();
  await Promise.resolve();
  reject(new Error("Go unavailable"));
  await failed;
  assert.equal(store.status().error, "Go unavailable");
  assert.equal(store.snapshot(), result);
  assert.equal(store.status().revision, 1);
  const retry = store.scan();
  await Promise.resolve();
  resolve(snapshot());
  await retry;
  assert.equal(store.status().error, null);
  assert.equal(store.status().revision, 2);
  store.dispose();
  await assert.rejects(store.scan(), /关闭/);
});

test("worker respects an already cancelled scan", async () => {
  const controller = new AbortController();
  controller.abort();
  await assert.rejects(
    runWorker(() => {}, controller.signal),
    /取消/,
  );
});

test("service API exposes live status while pending, coalesces refresh, exports and blocks foreign requests", async () => {
  let resolve;
  let runs = 0;
  const app = await startAtlasServer({
    port: 0,
    withVite: false,
    scanner: () => {
      runs++;
      return new Promise((yes) => {
        resolve = yes;
      });
    },
  });
  try {
    const status = await (await fetch(app.url + "/api/atlas/status")).json();
    assert.equal(status.scanning, true);
    assert.equal((await fetch(app.url + "/api/atlas/snapshot")).status, 503);
    assert.equal(
      (await fetch(app.url + "/api/atlas/scan", { method: "POST" })).status,
      403,
    );
    assert.equal(
      (
        await fetch(app.url + "/api/atlas/scan", {
          method: "POST",
          headers: { "X-Code-Atlas": "1", Origin: "https://example.com" },
        })
      ).status,
      403,
    );
    const foreignHostStatus = await new Promise((resolve, reject) => {
      http
        .get(
          app.url + "/api/atlas/status",
          { headers: { Host: "example.com" } },
          (response) => {
            response.resume();
            resolve(response.statusCode);
          },
        )
        .on("error", reject);
    });
    assert.equal(foreignHostStatus, 403);
    assert.equal(
      (
        await fetch(app.url + "/api/atlas/scan", {
          method: "POST",
          headers: { "X-Code-Atlas": "1" },
        })
      ).status,
      202,
    );
    assert.equal(runs, 1);
    const pending = app.store.scan();
    resolve(snapshot());
    await pending;
    const response = await (
      await fetch(app.url + "/api/atlas/snapshot")
    ).json();
    assert.equal(response.revision, 1);
    assert.equal(response.sessionId, status.sessionId);
    const exported = await fetch(app.url + "/api/atlas/export");
    assert.match(exported.headers.get("content-disposition"), /attachment/);
    const html = await exported.text();
    assert.match(html, /function mountAtlas/);
    assert.match(html, /代码地形/);
    assert.doesNotMatch(html, /__ATLAS_/);
    assert.equal(
      (await fetch(app.url + "/api/atlas/no-such-route")).status,
      404,
    );
    assert.equal((await fetch(app.url + "/api/atlas/scan")).status, 404);
  } finally {
    await app.close();
  }
});

test("offline export escapes script-closing metadata and ships the same executable explorer", () => {
  const data = snapshot();
  data.branch = "</script><script>unexpected()</script>";
  const html = renderOfflineHtml(data);
  assert.equal((html.match(/<script/g) || []).length, 2);
  const scripts = [...html.matchAll(/<script(?:[^>]*)>([\s\S]*?)<\/script>/g)];
  assert.equal(JSON.parse(scripts[0][1]).branch, data.branch);
  assert.doesNotThrow(() => new vm.Script(scripts[1][1]));
});
