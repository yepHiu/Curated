import assert from "node:assert/strict";
import { test } from "node:test";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { execFileSync } from "node:child_process";
import {
  analyzeScript,
  analyzeVue,
  category,
  resolveImport,
  root,
} from "./code-atlas.mjs";

test("AST ignores branch words in strings/comments and isolates nested functions", () => {
  const result = analyzeScript(
    "sample.ts",
    `function outer(a: boolean) {
    const text = 'if for while && ||'; // if (x) {}
    const inner = () => { if (a) return 1; return 0 }
    if (a && inner()) return 1
    return 0
  }`,
  );
  assert.deepEqual(result.errors, []);
  assert.deepEqual(
    result.functions.map((f) => [f.name, f.line, f.end, f.branches]),
    [
      ["outer", 1, 6, 3],
      ["inner", 3, 3, 2],
    ],
  );
});

test("Vue script offsets point to the original SFC and skip template expressions", () => {
  const result = analyzeVue(
    "sample.vue",
    `<template><button @click="() => 1">Hello</button></template>
<script setup lang="ts">
import Child from './Child.vue'
const act = () => true ? 1 : 0
</script>`,
  );
  assert.deepEqual(result.errors, []);
  assert.equal(result.functions.length, 1);
  assert.deepEqual(
    [
      result.functions[0].line,
      result.functions[0].end,
      result.functions[0].branches,
    ],
    [4, 4, 2],
  );
  assert.deepEqual(result.imports, ["./Child.vue"]);
});

test("parse failures are surfaced rather than accepted as accurate counts", () => {
  assert.ok(analyzeScript("bad.ts", "function broken( {").errors.length);
});

test("callback labels contain symbols rather than chained call arguments", () => {
  const result = analyzeScript("sample.ts", "load('private-value').then(() => 1)");
  assert.equal(result.functions[0].name, "then · 回调");
});

test("local imports resolve aliases, index modules, queries and emitted .js names", () => {
  const paths = new Set(["src/lib/a.ts", "src/ui/index.ts", "src/Child.vue"]);
  assert.equal(resolveImport("src/main.ts", "@/lib/a", paths), "src/lib/a.ts");
  assert.equal(
    resolveImport("src/main.ts", "./lib/a.js", paths),
    "src/lib/a.ts",
  );
  assert.equal(resolveImport("src/main.ts", "./ui", paths), "src/ui/index.ts");
  assert.equal(
    resolveImport("src/main.ts", "./Child.vue?raw", paths),
    "src/Child.vue",
  );
  assert.equal(resolveImport("src/main.ts", "vue", paths), null);
  assert.equal(category("src/lib/a.test.ts"), "test");
  assert.equal(category("backend/internal/a_test.go"), "test");
  assert.equal(category("scripts/release/tests/test_a.py"), "test");
  assert.equal(category("docs/plan/demo.html"), "docs");
  assert.equal(category("src/env.d.ts"), "config");
});

test("Go methods, closures, select/case branches and syntax errors retain positions", () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), "curated-atlas-test-"));
  try {
    const fixture = path.join(temp, "sample.go"),
      broken = path.join(temp, "broken.go");
    fs.writeFileSync(
      fixture,
      `package sample
type Thing struct{}
func (t *Thing) Run(a bool) {
  nested := func() { if a { return } }
  if a && true { nested() }
  switch { case a: nested(); default: }
  select { case <- make(chan int): nested(); default: }
}`,
    );
    fs.writeFileSync(broken, "package sample\nfunc broken(");
    const result = JSON.parse(
      execFileSync(
        "go",
        ["run", path.join(root, "scripts/analysis/go-atlas.go")],
        {
          cwd: path.join(root, "backend"),
          input: JSON.stringify([fixture, broken]),
          encoding: "utf8",
        },
      ),
    );
    assert.deepEqual(
      result[0].functions.map((f) => [f.name, f.line, f.end, f.branches]),
      [
        ["Thing.Run", 3, 8, 5],
        ["匿名函数", 4, 4, 2],
      ],
    );
    assert.ok(result[1].error);
  } finally {
    // Only remove the explicitly created fixture files and their empty temporary directory.
    for (const name of ["sample.go", "broken.go"])
      fs.rmSync(path.join(temp, name), { force: true });
    fs.rmdirSync(temp);
  }
});
