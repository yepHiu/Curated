import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { execFileSync } from "node:child_process";
import ts from "typescript";
import { parse as parseVue } from "vue/compiler-sfc";

const here = path.dirname(fileURLToPath(import.meta.url));
export const root = path.resolve(here, "../..");
const output = "docs/reference/code-atlas.html";
const sourceExtensions = new Set([
  ".ts",
  ".tsx",
  ".mts",
  ".cts",
  ".js",
  ".jsx",
  ".mjs",
  ".cjs",
  ".vue",
  ".go",
  ".py",
  ".c",
  ".h",
  ".cpp",
  ".mm",
  ".m",
  ".html",
  ".css",
  ".scss",
  ".sql",
  ".sh",
  ".ps1",
  ".iss",
  ".bat",
  ".cmd",
]);
const parsedExtensions = new Set([
  ".ts",
  ".tsx",
  ".mts",
  ".cts",
  ".js",
  ".jsx",
  ".mjs",
  ".cjs",
  ".vue",
  ".go",
]);
const ignored =
  /(^|\/)(node_modules|vendor|dist|frontend-dist|electron-dist|runtime|\.git|\.workspace|coverage|test-results|playwright-report)(\/|$)/;

export function category(p) {
  if (
    /\.(test|spec)\.[^/]+$|_test\.go$|(^|\/)(tests?|__tests__|testdata|fixtures)(\/|$)|(^|\/)test_[^/]+\.py$/.test(
      p,
    )
  )
    return "test";
  if (
    /^(docs|\.cursor|\.github\/ISSUE_TEMPLATE)\//.test(p) ||
    /\.(md|mdc|txt|rst)$/i.test(p)
  )
    return "docs";
  if (/\.d\.ts$/.test(p)) return "config";
  if (sourceExtensions.has(path.extname(p))) return "source";
  if (
    /\.(json|ya?ml|toml|cfg|ini|xml|lock)$|(^|\/)(Dockerfile|Makefile|go\.(mod|sum)|\.\w+)$/.test(
      p,
    )
  )
    return "config";
  return "asset";
}

export function moduleOf(p) {
  const parts = p.split("/");
  if (p.startsWith("backend/internal/")) return parts.slice(0, 3).join("/");
  if (p.startsWith("backend/")) return parts.slice(0, 2).join("/");
  if (p.startsWith("src/components/")) return parts.slice(0, 3).join("/");
  if (/^(src|scripts)\//.test(p)) return parts.slice(0, 2).join("/");
  return parts.length > 1 ? parts[0] : "(root)";
}

function isFunction(node) {
  return (
    (ts.isFunctionDeclaration(node) ||
      ts.isFunctionExpression(node) ||
      ts.isArrowFunction(node) ||
      ts.isMethodDeclaration(node) ||
      ts.isGetAccessorDeclaration(node) ||
      ts.isSetAccessorDeclaration(node) ||
      ts.isConstructorDeclaration(node)) &&
    !!node.body
  );
}

export function analyzeScript(p, text, offset = 0, jsx = /\.[jt]sx$/.test(p)) {
  const file = ts.createSourceFile(
    p,
    text,
    ts.ScriptTarget.Latest,
    true,
    jsx ? ts.ScriptKind.TSX : ts.ScriptKind.TS,
  );
  const functions = [],
    imports = [];
  function walk(node) {
    if (isFunction(node)) {
      let name = node.name?.getText(file);
      if (
        !name &&
        (ts.isVariableDeclaration(node.parent) ||
          ts.isPropertyAssignment(node.parent))
      )
        name = node.parent.name.getText(file);
      if (!name && ts.isCallExpression(node.parent)) {
        const callee = node.parent.expression;
        // Keep symbol labels, never arguments from a chained call expression.
        const label = ts.isIdentifier(callee)
          ? callee.text
          : ts.isPropertyAccessExpression(callee)
            ? callee.name.text
            : "匿名";
        name = `${label} · 回调`;
      }
      if (ts.isConstructorDeclaration(node)) name = "constructor";
      let branches = 1;
      function decisions(child) {
        if (isFunction(child)) return;
        if (
          ts.isIfStatement(child) ||
          ts.isForStatement(child) ||
          ts.isForInStatement(child) ||
          ts.isForOfStatement(child) ||
          ts.isWhileStatement(child) ||
          ts.isDoStatement(child) ||
          ts.isCaseClause(child) ||
          ts.isCatchClause(child) ||
          ts.isConditionalExpression(child)
        )
          branches++;
        if (
          ts.isBinaryExpression(child) &&
          [
            ts.SyntaxKind.AmpersandAmpersandToken,
            ts.SyntaxKind.BarBarToken,
            ts.SyntaxKind.QuestionQuestionToken,
          ].includes(child.operatorToken.kind)
        )
          branches++;
        ts.forEachChild(child, decisions);
      }
      decisions(node.body);
      functions.push({
        name: name || "匿名函数",
        line:
          file.getLineAndCharacterOfPosition(node.getStart(file)).line +
          1 +
          offset,
        end:
          file.getLineAndCharacterOfPosition(
            Math.max(node.getStart(file), node.end - 1),
          ).line +
          1 +
          offset,
        branches,
      });
    }
    if (
      (ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) &&
      node.moduleSpecifier &&
      ts.isStringLiteralLike(node.moduleSpecifier)
    )
      imports.push(node.moduleSpecifier.text);
    if (
      ts.isImportEqualsDeclaration(node) &&
      ts.isExternalModuleReference(node.moduleReference) &&
      node.moduleReference.expression &&
      ts.isStringLiteralLike(node.moduleReference.expression)
    )
      imports.push(node.moduleReference.expression.text);
    if (
      ts.isCallExpression(node) &&
      (node.expression.kind === ts.SyntaxKind.ImportKeyword ||
        (ts.isIdentifier(node.expression) &&
          node.expression.text === "require")) &&
      node.arguments[0] &&
      ts.isStringLiteralLike(node.arguments[0])
    )
      imports.push(node.arguments[0].text);
    ts.forEachChild(node, walk);
  }
  walk(file);
  return {
    functions,
    imports: [...new Set(imports)],
    errors: file.parseDiagnostics.map((d) =>
      ts.flattenDiagnosticMessageText(d.messageText, " "),
    ),
  };
}

export function analyzeVue(p, text) {
  const { descriptor, errors } = parseVue(text, { filename: p });
  const result = {
    functions: [],
    imports: [],
    errors: errors.map((e) => (typeof e === "string" ? e : e.message)),
  };
  for (const block of [descriptor.script, descriptor.scriptSetup].filter(
    Boolean,
  )) {
    if (block.src) result.imports.push(block.src);
    const analysis = analyzeScript(
      p,
      block.content,
      block.loc.start.line - 1,
      block.lang === "tsx" || block.lang === "jsx",
    );
    result.functions.push(...analysis.functions);
    result.imports.push(...analysis.imports);
    result.errors.push(...analysis.errors);
  }
  return result;
}

export function resolveImport(from, spec, paths) {
  const clean = spec.replace(/[?#].*$/, "");
  let base;
  if (clean.startsWith("@/")) base = `src/${clean.slice(2)}`;
  else if (clean.startsWith("."))
    base = path.posix.normalize(
      path.posix.join(path.posix.dirname(from), clean),
    );
  else return null;
  const bases = [base];
  if (/\.[mc]?js$/.test(base))
    bases.push(
      base
        .replace(/\.mjs$/, ".mts")
        .replace(/\.cjs$/, ".cts")
        .replace(/\.js$/, ".ts"),
    );
  for (const b of bases) {
    for (const suffix of [
      "",
      ".ts",
      ".tsx",
      ".js",
      ".jsx",
      ".mjs",
      ".cjs",
      ".vue",
      ".json",
      "/index.ts",
      "/index.js",
      "/index.vue",
    ]) {
      if (paths.has(b + suffix)) return b + suffix;
    }
  }
  return null;
}

function git(...args) {
  return execFileSync("git", args, {
    cwd: root,
    encoding: "utf8",
    maxBuffer: 20 * 1024 * 1024,
  });
}

export function generate() {
  const paths = [
    ...new Set(
      git("ls-files", "-z", "--cached", "--others", "--exclude-standard")
        .split("\0")
        .filter(Boolean),
    ),
  ].sort();
  const files = [],
    functions = [],
    warnings = [],
    excluded = [];
  const imports = new Map();
  for (const p of paths) {
    if (p === output || ignored.test(p)) {
      excluded.push(p);
      continue;
    }
    const absolute = path.join(root, p);
    if (!fs.existsSync(absolute)) {
      excluded.push(p);
      continue;
    }
    const stat = fs.lstatSync(absolute);
    if (!stat.isFile()) {
      excluded.push(p);
      continue;
    }
    const buffer = fs.readFileSync(absolute);
    const text = buffer.includes(0) ? null : buffer.toString("utf8");
    const ext = path.extname(p);
    const kind = category(p);
    const lines =
      text === null || !text
        ? 0
        : text.split(/\r\n|\n|\r/).length - (/\r?\n$|\r$/.test(text) ? 1 : 0);
    const f = {
      path: p,
      ext: ext || "(none)",
      kind,
      module: moduleOf(p),
      bytes: buffer.length,
      lines,
      nonblank: text?.split(/\r\n|\n|\r/).filter((l) => l.trim()).length || 0,
      functions: 0,
      maxBranches: 0,
      parsed: false,
    };
    if (
      text !== null &&
      parsedExtensions.has(ext) &&
      ext !== ".go" &&
      kind !== "docs" &&
      !p.endsWith(".d.ts")
    ) {
      const analysis =
        ext === ".vue" ? analyzeVue(p, text) : analyzeScript(p, text);
      if (analysis.errors.length)
        warnings.push({ path: p, message: analysis.errors.join("; ") });
      else {
        f.parsed = true;
        addFunctions(f, analysis.functions);
        imports.set(p, analysis.imports);
      }
    }
    files.push(f);
  }
  function addFunctions(file, items) {
    file.functions = items.length;
    file.maxBranches = items.reduce((max, fn) => Math.max(max, fn.branches), 0);
    functions.push(
      ...items.map((fn) => ({
        ...fn,
        lines: fn.end - fn.line + 1,
        path: file.path,
        kind: file.kind,
        module: file.module,
      })),
    );
  }
  const goFiles = files.filter((f) => f.ext === ".go" && f.kind !== "docs");
  if (goFiles.length) {
    // Go's system caches remain untouched; this helper uses only the standard library.
    const results = JSON.parse(
      execFileSync("go", ["run", path.join(here, "go-atlas.go")], {
        cwd: path.join(root, "backend"),
        input: JSON.stringify(goFiles.map((f) => path.join(root, f.path))),
        encoding: "utf8",
        maxBuffer: 50 * 1024 * 1024,
      }),
    );
    results.forEach((result, i) => {
      if (result.error)
        warnings.push({ path: goFiles[i].path, message: result.error });
      else {
        goFiles[i].parsed = true;
        addFunctions(goFiles[i], result.functions);
        imports.set(goFiles[i].path, result.imports);
      }
    });
  }
  const fileSet = new Set(files.map((f) => f.path));
  const packageSet = new Set(goFiles.map((f) => path.posix.dirname(f.path)));
  const goModule = fs
    .readFileSync(path.join(root, "backend/go.mod"), "utf8")
    .match(/^module\s+(\S+)/m)[1];
  const edges = [],
    unresolved = [];
  for (const [from, specs] of imports) {
    for (const spec of new Set(specs)) {
      if (from.endsWith(".go")) {
        if (spec.startsWith(`${goModule}/`)) {
          const to = `backend/${spec.slice(goModule.length + 1)}`;
          if (packageSet.has(to))
            edges.push({
              from,
              to,
              type: "package",
              source: moduleOf(from),
              target: moduleOf(`${to}/_.go`),
            });
          else unresolved.push({ from, spec });
        }
      } else {
        const to = resolveImport(from, spec, fileSet);
        if (to)
          edges.push({
            from,
            to,
            type: "file",
            source: moduleOf(from),
            target: moduleOf(to),
          });
        else if (spec.startsWith(".") || spec.startsWith("@/"))
          unresolved.push({ from, spec });
      }
    }
  }
  const data = {
    schema: 1,
    generatedAt: new Date().toISOString(),
    commit: git("rev-parse", "--short", "HEAD").trim(),
    branch: git("branch", "--show-current").trim() || "(detached)",
    dirty: !!git("status", "--porcelain").trim(),
    files,
    functions,
    edges,
    warnings,
    unresolved,
    excluded,
    counts: {
      files: files.length,
      source: files.filter((f) => f.kind === "source").length,
      sourceLines: files
        .filter((f) => f.kind === "source")
        .reduce((n, f) => n + f.lines, 0),
      functions: functions.length,
      parsedFiles: files.filter((f) => f.parsed).length,
    },
  };
  const template = fs.readFileSync(
    path.join(here, "code-atlas.template.html"),
    "utf8",
  );
  const json = JSON.stringify(data)
    .replace(/</g, "\\u003c")
    .replace(/\u2028/g, "\\u2028")
    .replace(/\u2029/g, "\\u2029");
  fs.writeFileSync(
    path.join(root, output),
    template.replace("/*__ATLAS_DATA__*/", () => json),
  );
  console.log(
    JSON.stringify(
      {
        output,
        ...data.counts,
        edges: edges.length,
        warnings,
        unresolved: unresolved.length,
      },
      null,
      2,
    ),
  );
  return data;
}

if (
  process.argv[1] &&
  path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)
)
  generate();
