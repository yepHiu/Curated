import { test } from "node:test";
import assert from "node:assert/strict";
import { buildFileGraph, zoomCamera, fitGraph } from "./app/network-model.mjs";

const file = (path, module = "src/lib") => ({
  path,
  ext: path.slice(path.lastIndexOf(".")),
  module,
  lines: 10,
});

test("file network retains every file including isolated files and same-module imports", () => {
  const files = Array.from({ length: 500 }, (_, i) =>
    file(`src/lib/file-${i}.ts`),
  );
  const data = {
    files,
    edges: [
      {
        from: files[0].path,
        to: files[1].path,
        type: "file",
        source: "src/lib",
        target: "src/lib",
      },
    ],
  };
  const graph = buildFileGraph(data, files);
  assert.equal(graph.nodes.length, 500);
  assert.equal(graph.edges.length, 1);
  assert.equal(graph.byId.get(files[0].path).outgoing[0], files[1].path);
  assert.equal(graph.byId.get(files[1].path).incoming[0], files[0].path);
  assert.equal(graph.byId.get(files[499].path).degree, 0);
  assert(
    graph.nodes.every(
      (node) => Number.isFinite(node.x) && Number.isFinite(node.y),
    ),
  );
  const second = buildFileGraph(data, files);
  assert.deepEqual(
    graph.nodes.map((n) => [n.x, n.y]),
    second.nodes.map((n) => [n.x, n.y]),
  );
});

test("Go imports target packages, membership never fabricates file-level calls", () => {
  const files = [
    file("backend/internal/a/run.go", "backend/internal/a"),
    file("backend/internal/b/first.go", "backend/internal/b"),
    file("backend/internal/b/second.go", "backend/internal/b"),
  ];
  const graph = buildFileGraph(
    {
      files,
      edges: [
        { from: files[0].path, to: "backend/internal/b", type: "package" },
      ],
    },
    files,
  );
  assert.equal(graph.nodes.length, 5);
  assert.equal(graph.edges.filter((e) => e.kind === "import").length, 1);
  assert.equal(graph.edges.filter((e) => e.kind === "membership").length, 3);
  assert.equal(graph.byId.get(files[1].path).incoming.length, 0);
  assert.equal(graph.byId.get("package:backend/internal/b").incoming.length, 1);
  assert.equal(graph.byId.get("package:backend/internal/b").members.length, 2);
});

test("scope includes direct external-scope targets but not unrelated files, and de-duplicates edges", () => {
  const files = [
    file("src/a.ts"),
    file("src/shared.ts"),
    file("src/unrelated.ts"),
  ];
  const edge = { from: files[0].path, to: files[1].path, type: "file" };
  const graph = buildFileGraph(
    { files, edges: [edge, edge] },
    files.slice(0, 1),
  );
  assert.equal(graph.nodes.length, 2);
  assert.equal(graph.byId.get("src/shared.ts").context, true);
  assert.equal(graph.edges.length, 1);
});

test("zoom preserves the world coordinate under the pointer and clamps scale", () => {
  const camera = { x: 50, y: -80, scale: 0.5 },
    p = { x: 320, y: 200 };
  for (const factor of [0.01, 0.5, 2, 100]) {
    const next = zoomCamera(camera, factor, p);
    assert(next.scale >= 0.025 && next.scale <= 8);
    assert(
      Math.abs((p.x - camera.x) / camera.scale - (p.x - next.x) / next.scale) <
        1e-9,
    );
    assert(
      Math.abs((p.y - camera.y) / camera.scale - (p.y - next.y) / next.scale) <
        1e-9,
    );
  }
});

test("fit centers the complete graph within viewport and empty graphs remain navigable", () => {
  const bounds = { x: -500, y: -200, width: 2000, height: 1000 },
    camera = fitGraph(bounds, 1000, 700);
  assert.equal((bounds.x + bounds.width / 2) * camera.scale + camera.x, 500);
  assert.equal((bounds.y + bounds.height / 2) * camera.scale + camera.y, 350);
  assert(bounds.width * camera.scale <= 1000);
  assert(bounds.height * camera.scale <= 700);
  const empty = buildFileGraph({ files: [], edges: [] }, []);
  assert.equal(empty.nodes.length, 0);
  assert(Number.isFinite(fitGraph(empty.bounds, 300, 400).scale));
});
