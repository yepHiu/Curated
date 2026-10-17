import { mountFileNetwork } from "./network-canvas.mjs";
// Shared visual explorer, isolated by its host; used by Vue and offline exports.
export function mountAtlas(host, D, previousState = {}, live = false) {
  const controller = new AbortController();
  const listen = (target, type, handler) =>
    target.addEventListener(type, handler, { signal: controller.signal });

  // Fragment navigation cannot target an element inside a shadow root.
  const skipLink = host.querySelector(".skip");
  listen(skipLink, "click", (event) => {
    event.preventDefault();
    const main = host.querySelector("#main");
    main.tabIndex = -1;
    main.focus();
    main.scrollIntoView({ block: "start" });
  });

  ("use strict");

  const $ = (s) => host.querySelector(s),
    $$ = (s) => [...host.querySelectorAll(s)];
  const esc = (s) =>
    String(s).replace(
      /[&<>"']/g,
      (c) =>
        ({
          "&": "&amp;",
          "<": "&lt;",
          ">": "&gt;",
          '"': "&quot;",
          "'": "&#39;",
        })[c],
    );
  const num = (n) => n.toLocaleString("zh-CN"),
    sum = (xs, k) => xs.reduce((n, x) => n + (x[k] || 0), 0),
    pct = (n, d) => (d ? ((100 * n) / d).toFixed(1) + "%" : "0%");
  const bytes = (n) =>
    n >= 1048576
      ? (n / 1048576).toFixed(1) + " MiB"
      : n >= 1024
        ? (n / 1024).toFixed(1) + " KiB"
        : n + " B";
  const colors = {
    backend: ["#1c3d40", "#a2e6d2"],
    src: ["#233955", "#bad6ff"],
    electron: ["#3e304c", "#e5c9f5"],
    scripts: ["#443b2b", "#f2d5a4"],
    docs: ["#2e344d", "#c9cef2"],
    other: ["#2b3946", "#c4d4e4"],
  };
  const kindNames = {
    source: "实现源码",
    test: "测试",
    docs: "文档",
    config: "配置 / 声明",
    asset: "资源 / 其他",
  };
  const state = {
    view: "map",
    scope: "source",
    query: "",
    queryInput: "",
    dir: "",
    metric: "lines",
    sort: "lines",
    page: 0,
    ...previousState,
  };
  const fileMap = new Map(D.files.map((f) => [f.path, f]));
  const functionMap = new Map();
  for (const f of D.functions) {
    if (!functionMap.has(f.path)) functionMap.set(f.path, []);
    functionMap.get(f.path).push(f);
  }
  function palette(p) {
    return colors[p.split("/")[0]] || colors.other;
  }
  function inScope(f) {
    return (
      state.scope === "all" ||
      f.kind === state.scope ||
      (state.scope === "code" && (f.kind === "source" || f.kind === "test"))
    );
  }
  function filteredFiles() {
    return D.files.filter(
      (f) =>
        inScope(f) &&
        (!state.query ||
          f.path.toLowerCase().includes(state.query) ||
          (functionMap.get(f.path) || []).some((fn) =>
            fn.name.toLowerCase().includes(state.query),
          )),
    );
  }
  function filteredFunctions() {
    return D.functions.filter(
      (f) =>
        inScope(f) &&
        (!state.query ||
          (f.path + " " + f.name).toLowerCase().includes(state.query)),
    );
  }
  function group(xs, key) {
    const m = new Map();
    for (const x of xs) {
      const k = x[key];
      if (!m.has(k)) m.set(k, []);
      m.get(k).push(x);
    }
    return [...m].map(([name, items]) => ({
      name,
      items,
      lines: sum(items, "lines"),
      bytes: sum(items, "bytes"),
      functions: sum(items, "functions"),
      count: items.length,
    }));
  }
  function fileButton(f, label) {
    return (
      '<button class="path-button" data-file="' +
      esc(f.path) +
      '">' +
      esc(label || f.path) +
      "</button>"
    );
  }
  function announce(text) {
    $("#toast").textContent = text;
    $("#toast").hidden = false;
    clearTimeout(announce.timer);
    announce.timer = setTimeout(() => ($("#toast").hidden = true), 2600);
  }
  const source = D.files.filter((f) => f.kind === "source"),
    tests = D.files.filter((f) => f.kind === "test"),
    sourceFunctions = D.functions.filter((f) => f.kind === "source");
  $("#snapshot").innerHTML =
    "<strong>" +
    esc(
      new Date(D.generatedAt).toLocaleString("zh-CN", {
        timeZone: "Asia/Shanghai",
        hour12: false,
      }),
    ) +
    ' CST</strong><br><span class="mono">' +
    esc(D.commit) +
    "</span> · " +
    (D.dirty ? "含工作区修改" : "已提交工作区") +
    "<br>" +
    esc(D.branch);
  $("#metrics").innerHTML = [
    [
      "仓库文件",
      num(D.files.length),
      num(source.length) + " 个源码文件 · " + num(tests.length) + " 个测试文件",
    ],
    [
      "实现源码 · 物理行",
      num(sum(source, "lines")),
      "另有 " + num(sum(tests, "lines")) + " 行测试文件",
    ],
    [
      "已识别源码函数",
      num(sourceFunctions.length),
      "JS / TS / Vue script / Go · 包含回调",
    ],
    [
      "仓库文件内容",
      bytes(sum(D.files, "bytes")),
      "包括文档与资源 · 不含本地构建产物",
    ],
  ]
    .map(
      ([label, value, hint]) =>
        '<div class="metric"><label>' +
        label +
        "</label><strong>" +
        value +
        "</strong><small>" +
        hint +
        "</small></div>",
    )
    .join("");
  $("#diagnostics").innerHTML =
    "<p>解析成功 " +
    num(D.counts.parsedFiles) +
    " 个文件；解析失败 " +
    num(D.warnings.length) +
    " 个；未解析的本地导入 " +
    num(D.unresolved.length) +
    " 条；额外排除 " +
    num(D.excluded.length) +
    " 个 Git 候选文件。忽略目录未遍历，因此不提供其文件数量。</p>" +
    (D.warnings.length || D.unresolved.length
      ? '<details><summary>解析与导入诊断</summary><pre style="white-space:pre-wrap;overflow-wrap:anywhere">' +
        esc(
          JSON.stringify(
            { warnings: D.warnings, unresolved: D.unresolved },
            null,
            2,
          ),
        ) +
        "</pre></details>"
      : "") +
    '<details><summary>排除文件</summary><div class="detail-list">' +
    D.excluded.map(esc).join("<br>") +
    "</div></details>";

  let network = null;
  function render() {
    network?.destroy();
    network = null;
    $$(".tabs button").forEach((b) =>
      b.setAttribute("aria-pressed", String(b.dataset.view === state.view)),
    );
    if (state.view === "map") renderMap();
    else if (state.view === "functions") renderFunctions();
    else if (state.view === "network") renderNetwork();
    else renderFiles();
  }
  function breadcrumb() {
    let path = "";
    return (
      '<button data-dir="">仓库</button>' +
      state.dir
        .split("/")
        .filter(Boolean)
        .map((part) => {
          path += part + "/";
          return (
            '<span class="muted">/</span><button data-dir="' +
            esc(path) +
            '">' +
            esc(part) +
            "</button>"
          );
        })
        .join("")
    );
  }
  function mapChildren(files) {
    const grouped = new Map();
    for (const f of files) {
      if (!f.path.startsWith(state.dir)) continue;
      const rest = f.path.slice(state.dir.length),
        slash = rest.indexOf("/"),
        directory = slash !== -1,
        name = directory ? rest.slice(0, slash) : rest,
        key = state.dir + name + (directory ? "/" : "");
      if (!grouped.has(key))
        grouped.set(key, {
          path: key,
          name,
          directory,
          lines: 0,
          bytes: 0,
          functions: 0,
          count: 0,
        });
      const g = grouped.get(key);
      for (const k of ["lines", "bytes", "functions"]) g[k] += f[k];
      g.count++;
    }
    return [...grouped.values()].sort(
      (a, b) =>
        b[state.metric] - a[state.metric] || a.path.localeCompare(b.path),
    );
  }
  function rectangles(items, x, y, w, h, out = []) {
    if (!items.length) return out;
    if (items.length === 1) {
      out.push({ ...items[0], x, y, w, h });
      return out;
    }
    const total = sum(items, state.metric);
    let running = 0,
      index = 1,
      best = Infinity;
    for (let i = 1; i < items.length; i++) {
      running += items[i - 1][state.metric];
      const delta = Math.abs(total / 2 - running);
      if (delta < best) {
        best = delta;
        index = i;
      }
    }
    const a = items.slice(0, index),
      b = items.slice(index),
      ratio = sum(a, state.metric) / total;
    if (w >= h) {
      rectangles(a, x, y, w * ratio, h, out);
      rectangles(b, x + w * ratio, y, w * (1 - ratio), h, out);
    } else {
      rectangles(a, x, y, w, h * ratio, out);
      rectangles(b, x, y + h * ratio, w, h * (1 - ratio), out);
    }
    return out;
  }
  function renderMap() {
    const files = filteredFiles(),
      local = files.filter((f) => f.path.startsWith(state.dir)),
      children = mapChildren(files),
      ranked = [...local].sort((a, b) => b.lines - a.lines),
      funcs = local.flatMap((f) => functionMap.get(f.path) || []),
      total = sum(local, "lines"),
      top = ranked[0];
    const supported = local.filter((f) => f.parsed),
      sourceLocal = local.filter((f) => ["source", "test"].includes(f.kind));
    $("#view").innerHTML =
      '<div class="grid"><section class="panel"><div class="panel-head"><div><h2>代码的地形</h2><p id="map-summary">' +
      num(local.length) +
      " 个文件 · " +
      num(total) +
      " 行 · " +
      bytes(sum(local, "bytes")) +
      '</p></div><label class="field">面积代表<select id="metric"><option value="lines">物理行数</option><option value="bytes">文件字节</option><option value="functions">已识别函数数</option></select></label></div><div class="crumbs">' +
      breadcrumb() +
      '</div><div class="map" id="treemap" aria-label="可下钻目录面积图"></div><div class="legend">' +
      Object.entries(colors)
        .map(
          ([k, c]) =>
            '<span><i style="background:' +
            c[1] +
            '"></i>' +
            {
              src: "Web 前端",
              backend: "Go 后端",
              electron: "桌面端",
              scripts: "工程脚本",
              docs: "文档",
              other: "其他",
            }[k] +
            "</span>",
        )
        .join("") +
      '</div><p class="caption">每块面积与数值成比例。点击目录继续探索，点击文件查看函数与依赖；颜色标识顶层目录。零值项仍保留在下方清单。</p></section><aside class="panel insights"><h2>这一层，值得留意</h2><div class="insight"><strong>体量集中度</strong><div class="number">' +
      pct(sum(ranked.slice(0, 20), "lines"), total) +
      '</div><p>最大的 20 个文件，占当前范围的行数比例。集中度帮助你找到阅读入口。</p></div><div class="insight"><strong>超过 100 行的函数</strong><div class="number">' +
      num(funcs.filter((f) => f.lines > 100).length) +
      ' <span style="font-size:13px;color:var(--muted)">/ ' +
      num(funcs.length) +
      '</span></div><p>按函数跨度计算，包含匿名回调；大于 100 行是观察阈值，不是质量判决。</p></div><div class="insight"><strong>当前最大的文件</strong><p class="mono">' +
      (top ? esc(top.path) : "没有匹配的文件") +
      '</p><div class="number">' +
      (top ? num(top.lines) : "—") +
      ' <span style="font-size:13px;color:var(--muted)">行</span></div><p>函数解析覆盖 ' +
      num(supported.length) +
      " / " +
      num(sourceLocal.length) +
      ' 个源码或测试文件。</p></div></aside></div><section class="panel section-gap"><div class="subhead"><h2>目录切片</h2><small>与上方面积图同步 · ' +
      num(children.length) +
      ' 项</small></div><div class="table-wrap"><table><thead><tr><th>目录 / 文件</th><th class="num">文件数</th><th class="num">物理行</th><th class="num">函数</th><th class="num">字节</th></tr></thead><tbody>' +
      children
        .map(
          (c) =>
            '<tr><td><button class="path-button mono" ' +
            (c.directory
              ? 'data-dir="' + esc(c.path) + '"'
              : 'data-file="' + esc(c.path) + '"') +
            ">" +
            esc(c.name) +
            (c.directory ? " / →" : "") +
            '</button></td><td class="num">' +
            num(c.count) +
            '</td><td class="num">' +
            num(c.lines) +
            '</td><td class="num">' +
            num(c.functions) +
            '</td><td class="num">' +
            bytes(c.bytes) +
            "</td></tr>",
        )
        .join("") +
      "</tbody></table>" +
      (children.length
        ? ""
        : '<div class="empty">这里没有匹配的文件。试试清空搜索或返回上一级。</div>') +
      "</div></section>" +
      (state.dir ? "" : breakdown(files));
    $("#metric").value = state.metric;
    $("#metric").onchange = (e) => {
      state.metric = e.target.value;
      renderMap();
    };
    drawMap(children);
  }
  function drawMap(children) {
    const el = $("#treemap");
    if (!el) return;
    const active = children.filter((c) => c[state.metric] > 0),
      w = el.clientWidth,
      h = el.clientHeight;
    el.innerHTML = active.length
      ? rectangles(active, 0, 0, w, h)
          .map((c) => {
            const [bg, fg] = palette(c.path),
              tiny = c.w < 105 || c.h < 72,
              value =
                state.metric === "bytes"
                  ? bytes(c.bytes)
                  : num(c[state.metric]);
            return (
              '<button class="tile ' +
              (tiny ? "tiny" : "") +
              '" style="left:' +
              c.x +
              "px;top:" +
              c.y +
              "px;width:" +
              c.w +
              "px;height:" +
              c.h +
              "px;--tile-bg:" +
              bg +
              ";--tile-text:" +
              fg +
              '" ' +
              (c.directory
                ? 'data-dir="' + esc(c.path) + '"'
                : 'data-file="' + esc(c.path) + '"') +
              ' title="' +
              esc(c.path) +
              " · " +
              value +
              '" aria-label="' +
              esc(c.path) +
              "，" +
              value +
              "，" +
              (c.directory ? "进入目录" : "查看文件") +
              '"><strong>' +
              esc(c.name) +
              '</strong><span class="amount">' +
              value +
              "</span><small>" +
              num(c.count) +
              " 个文件" +
              (state.metric === "lines" ? " · 行" : "") +
              '</small><span class="arrow" aria-hidden="true">↗</span></button>'
            );
          })
          .join("")
      : '<div class="empty">当前范围没有可用于此面积指标的数据。<br>可以切换面积指标，或查看下方文件清单。</div>';
  }
  function breakdown(files) {
    const exts = group(files, "ext")
        .sort((a, b) => b.lines - a.lines)
        .slice(0, 7),
      kinds = group(D.files, "kind").sort((a, b) => b.bytes - a.bytes),
      totalLines = sum(files, "lines"),
      totalBytes = sum(D.files, "bytes");
    return (
      '<section class="panel section-gap breakdown"><div><h2>语言与文件类型</h2><p class="caption">当前筛选 · 按物理行数</p>' +
      exts
        .map(
          (g) =>
            '<div class="barrow"><span class="mono">' +
            esc(g.name) +
            '</span><div class="bar"><i style="width:' +
            pct(g.lines, totalLines) +
            '"></i></div><span style="text-align:right">' +
            pct(g.lines, totalLines) +
            "</span></div>",
        )
        .join("") +
      '</div><div><h2>仓库装着什么</h2><p class="caption">整个快照 · 按字节数，含文档与二进制资源</p>' +
      kinds
        .map(
          (g) =>
            '<div class="barrow"><span>' +
            kindNames[g.name] +
            '</span><div class="bar"><i style="width:' +
            pct(g.bytes, totalBytes) +
            ';background:var(--blue)"></i></div><span style="text-align:right">' +
            pct(g.bytes, totalBytes) +
            "</span></div>",
        )
        .join("") +
      "</div></section>"
    );
  }
  function pager(total) {
    const pages = Math.max(1, Math.ceil(total / 30));
    state.page = Math.min(state.page, pages - 1);
    return (
      '<div class="pager"><span>共 ' +
      num(total) +
      " 项 · 每页 30 项 · " +
      (state.page + 1) +
      " / " +
      pages +
      '</span><div><button data-page="-1" ' +
      (state.page === 0 ? "disabled" : "") +
      '>上一页</button><button data-page="1" ' +
      (state.page >= pages - 1 ? "disabled" : "") +
      ">下一页</button></div></div>"
    );
  }
  function renderFunctions() {
    const functions = filteredFunctions(),
      sort = ["branches", "lines", "name"].includes(state.sort)
        ? state.sort
        : "lines";
    functions.sort((a, b) =>
      sort === "name"
        ? a.name.localeCompare(b.name)
        : b[sort] - a[sort] || b.lines - a.lines,
    );
    const pagination = pager(functions.length),
      rows = functions.slice(state.page * 30, (state.page + 1) * 30),
      bins = [
        [1, 20],
        [21, 50],
        [51, 100],
        [101, Infinity],
      ];
    $("#view").innerHTML =
      '<section class="panel"><div class="panel-head"><div><h2>把镜头，推到函数级别</h2><p>' +
      num(functions.length) +
      ' 个匹配函数 · 含匿名回调 · 长度与分支分别观察</p></div><label class="field">排序<select id="sort"><option value="lines">函数长度 ↓</option><option value="branches">分支指标 ↓</option><option value="name">函数名称 A–Z</option></select></label></div><div class="distributions">' +
      bins
        .map(([min, max]) => {
          const n = functions.filter(
            (f) => f.lines >= min && f.lines <= max,
          ).length;
          return (
            '<div class="distribution"><small>' +
            (max === Infinity ? "超过 100" : min + "–" + max) +
            " 行</small><strong>" +
            num(n) +
            '</strong><div class="bar"><i style="width:' +
            pct(n, functions.length) +
            '"></i></div><small>' +
            pct(n, functions.length) +
            " 的函数</small></div>"
          );
        })
        .join("") +
      '</div><div class="table-wrap"><table><thead><tr><th>函数 / 位置</th><th>类型</th><th class="num">跨度</th><th class="num">分支指标</th></tr></thead><tbody>' +
      rows
        .map(
          (f) =>
            "<tr><td>" +
            fileButton(f, f.name) +
            '<span class="path-line">' +
            esc(f.path) +
            ":" +
            f.line +
            "–" +
            f.end +
            '</span></td><td><span class="badge">' +
            kindNames[f.kind] +
            '</span></td><td class="num">' +
            num(f.lines) +
            ' 行</td><td class="num"><span class="badge ' +
            (f.branches > 20 ? "warm" : "") +
            '">' +
            num(f.branches) +
            "</span></td></tr>",
        )
        .join("") +
      "</tbody></table>" +
      (rows.length
        ? ""
        : '<div class="empty">没有匹配的函数。试试其他关键词或范围。</div>') +
      "</div>" +
      pagination +
      '<p class="caption">函数跨度包含内部嵌套回调的代码；分支指标不重复计算嵌套函数的分支。点击函数名称查看所在文件的完整函数列表。</p></section>';
    $("#sort").value = sort;
    $("#sort").onchange = (e) => {
      state.sort = e.target.value;
      state.page = 0;
      renderFunctions();
    };
  }
  function renderFiles() {
    const files = filteredFiles(),
      sort = ["lines", "bytes", "functions", "maxBranches", "path"].includes(
        state.sort,
      )
        ? state.sort
        : "lines";
    files.sort((a, b) =>
      sort === "path" ? a.path.localeCompare(b.path) : b[sort] - a[sort],
    );
    const pagination = pager(files.length),
      rows = files.slice(state.page * 30, (state.page + 1) * 30);
    $("#view").innerHTML =
      '<section class="panel"><div class="panel-head"><div><h2>每一份文件，都有自己的重量</h2><p>' +
      num(files.length) +
      " 个文件 · " +
      num(sum(files, "lines")) +
      " 行 · " +
      bytes(sum(files, "bytes")) +
      '</p></div><label class="field">排序<select id="sort"><option value="lines">物理行数 ↓</option><option value="bytes">文件字节 ↓</option><option value="functions">函数数量 ↓</option><option value="maxBranches">最高分支指标 ↓</option><option value="path">路径 A–Z</option></select></label></div><div class="table-wrap"><table><thead><tr><th>文件</th><th>分类</th><th class="num">物理行</th><th class="num">函数</th><th class="num">字节</th></tr></thead><tbody>' +
      rows
        .map(
          (f) =>
            "<tr><td>" +
            fileButton(f) +
            '<span class="path-line">' +
            esc(f.module) +
            '</span></td><td><span class="badge">' +
            kindNames[f.kind] +
            '</span></td><td class="num">' +
            num(f.lines) +
            '</td><td class="num">' +
            (f.parsed ? num(f.functions) : "—") +
            '</td><td class="num">' +
            bytes(f.bytes) +
            "</td></tr>",
        )
        .join("") +
      "</tbody></table>" +
      (rows.length ? "" : '<div class="empty">没有匹配的文件。</div>') +
      "</div>" +
      pagination +
      "</section>";
    $("#sort").value = sort;
    $("#sort").onchange = (e) => {
      state.sort = e.target.value;
      state.page = 0;
      renderFiles();
    };
  }
  function renderNetwork() {
    network?.destroy();
    const files = D.files.filter(inScope);
    network = mountFileNetwork($("#view"), D, files, {
      query: state.queryInput,
      saved: state.graph,
      onState: (graph) => {
        state.graph = { camera: { ...graph.camera }, selected: graph.selected };
      },
      onFile: showFile,
    });
  }
  function showFile(p) {
    const file = fileMap.get(p);
    if (!file) return;
    const functions = [...(functionMap.get(p) || [])].sort(
        (a, b) => a.line - b.line,
      ),
      out = D.edges.filter((e) => e.from === p),
      incoming = D.edges.filter((e) =>
        e.type === "file"
          ? e.to === p
          : e.to === p.slice(0, p.lastIndexOf("/")),
      ),
      error = D.warnings.find((w) => w.path === p);
    $("#detail-content").innerHTML =
      '<div class="dialog-top"><div><div class="eyebrow">FILE INSPECTOR</div><h2 id="detail-title">' +
      esc(p) +
      "</h2><small>" +
      kindNames[file.kind] +
      " · " +
      esc(file.module) +
      '</small></div><button id="close-detail" autofocus>关闭</button></div><div class="dialog-stats"><span>' +
      num(file.lines) +
      " 物理行</span><span>" +
      num(file.nonblank) +
      " 非空行</span><span>" +
      bytes(file.bytes) +
      "</span><span>" +
      (file.parsed ? file.functions + " 个函数" : "未进行函数统计") +
      '</span></div><button id="copy-path">复制相对路径</button><p class="caption">路径与行号可用于编辑器定位。报告只保存统计和符号名称，不内嵌源码正文。</p><h3 style="margin-top:25px">函数清单 · 按出现顺序</h3>' +
      (functions.length
        ? '<div class="table-wrap"><table><thead><tr><th>函数</th><th class="num">起止行</th><th class="num">跨度</th><th class="num">分支</th></tr></thead><tbody>' +
          functions
            .map(
              (f) =>
                '<tr><td class="mono">' +
                esc(f.name) +
                '</td><td class="num">' +
                f.line +
                "–" +
                f.end +
                '</td><td class="num">' +
                f.lines +
                '</td><td class="num">' +
                f.branches +
                "</td></tr>",
            )
            .join("") +
          "</tbody></table></div>"
        : '<p class="caption">' +
          (error
            ? "解析失败：" + esc(error.message)
            : file.parsed
              ? "此文件没有带函数体的函数。"
              : "此文件的语言或类别不在函数解析范围内。") +
          "</p>") +
      '<div class="breakdown section-gap"><div><h3>本地导入 · ' +
      out.length +
      '</h3><div class="detail-list">' +
      (out
        .map((e) =>
          e.type === "file"
            ? fileButton({ path: e.to })
            : esc(e.to) + " <small>（Go 包）</small><br>",
        )
        .join("") || '<span class="muted">无已解析的本地导入</span>') +
      "</div></div><div><h3>" +
      (p.endsWith(".go") ? "所在 Go 包被引用" : "被其他文件引用") +
      " · " +
      incoming.length +
      '</h3><div class="detail-list">' +
      (incoming.map((e) => fileButton({ path: e.from })).join("") ||
        '<span class="muted">无已解析的本地引用</span>') +
      "</div></div></div>";
    $("#close-detail").onclick = () => $("#detail").close();
    $("#copy-path").onclick = async () => {
      try {
        await navigator.clipboard.writeText(p);
        announce("已复制路径");
      } catch {
        const area = document.createElement("textarea");
        area.value = p;
        $("#detail-content").append(area);
        area.select();
        const success = document.execCommand("copy");
        area.remove();
        announce(success ? "已复制路径" : "复制不可用，请直接选择上方路径");
      }
    };
    if (!$("#detail").open) $("#detail").showModal();
  }
  listen(host, "click", (e) => {
    const f = e.target.closest("[data-file]"),
      dir = e.target.closest("[data-dir]"),
      tab = e.target.closest("[data-view]"),
      page = e.target.closest("[data-page]");
    if (f) showFile(f.dataset.file);
    else if (dir) {
      state.dir = dir.dataset.dir;
      renderMap();
    } else if (tab) {
      state.view = tab.dataset.view;
      state.page = 0;
      state.sort = "lines";
      render();
    } else if (page) {
      state.page += Number(page.dataset.page);
      render();
    }
  });
  $("#search").addEventListener("input", (e) => {
    state.queryInput = e.target.value;
    state.query = e.target.value.trim().toLowerCase();
    state.page = 0;
    if (network) network.setQuery(state.queryInput);
    else render();
  });
  $("#scope").onchange = (e) => {
    state.scope = e.target.value;
    state.graph = undefined;
    state.page = 0;
    render();
  };
  $("#reset").onclick = () => {
    Object.assign(state, {
      scope: "source",
      query: "",
      queryInput: "",
      dir: "",
      metric: "lines",
      sort: "lines",
      page: 0,
      graph: undefined,
    });
    $("#search").value = "";
    $("#scope").value = "source";
    render();
  };
  let resizeTimer;
  listen(window, "resize", () => {
    clearTimeout(resizeTimer);
    resizeTimer = setTimeout(() => {
      if (state.view === "map") drawMap(mapChildren(filteredFiles()));
    }, 100);
  });
  $("#search").value = state.queryInput;
  $("#scope").value = state.scope;
  if (live) {
    $(".topbar").remove();
    $(".foot code").textContent = "pnpm code:atlas";
  }
  render();
  return {
    state,
    destroy() {
      network?.destroy();
      controller.abort();
      clearTimeout(resizeTimer);
      clearTimeout(announce.timer);
      if ($("#detail").open) $("#detail").close();
    },
  };
}
