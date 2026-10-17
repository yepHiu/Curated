import { buildFileGraph, zoomCamera, fitGraph } from "./network-model.mjs";

export function mountFileNetwork(container, data, files, options = {}) {
  const graph = buildFileGraph(data, files);
  const escape = (value) =>
    String(value).replace(
      /[&<>"']/g,
      (char) =>
        ({
          "&": "&amp;",
          "<": "&lt;",
          ">": "&gt;",
          '"': "&quot;",
          "'": "&#39;",
        })[char],
    );
  const number = (value) => value.toLocaleString("zh-CN");
  const imports = graph.edges.filter((edge) => edge.kind === "import").length;
  container.innerHTML = `<section class="graph-workspace">
    <div class="graph-heading"><div><h2>进入代码的网络</h2><p><strong>${number(graph.nodes.filter((n) => n.kind === "file").length)}</strong> 个文件 · ${number(graph.nodes.filter((n) => n.kind === "package").length)} 个 Go 包 · <strong>${number(imports)}</strong> 条导入 · 不截取 Top N</p></div><button data-graph-action="fullscreen">展开画布</button></div>
    <div class="graph-body"><div class="graph-stage">
      <canvas class="graph-canvas" tabindex="0" role="img" aria-label="完整文件依赖图。拖动平移，滚轮缩放，方向键移动，加减键缩放。使用右侧节点列表进行键盘选择。"></canvas>
      <div class="graph-tools"><button data-graph-action="in" aria-label="放大依赖图">＋</button><button data-graph-action="out" aria-label="缩小依赖图">−</button><button data-graph-action="fit">适应全图</button><output class="graph-zoom">100%</output></div>
      <div class="graph-hint">拖动画布 · 滚轮 / 双指缩放 · 点击节点查看连接</div>
      <div class="graph-minimap"><canvas aria-label="全图缩略图，点击定位"></canvas></div>
      <div class="graph-hover" hidden></div>
      ${graph.nodes.length ? "" : '<div class="graph-empty">当前范围没有文件，请切换统计范围。</div>'}
    </div><aside class="graph-inspector"><div class="graph-selection"></div><label class="field">定位文件或 Go 包<input class="graph-search" type="search" placeholder="输入路径，如 PlayerPage.vue" autocomplete="off" /></label><div class="graph-search-summary" role="status"></div><div class="graph-results"></div><button class="graph-more" hidden>显示更多匹配节点</button></aside></div>
    <div class="graph-footer"><span><i style="background:#8fb9ff"></i>Web 前端</span><span><i style="background:#81dec0"></i>Go 后端</span><span><i style="background:#d4a4f2"></i>桌面端</span><span><i style="background:#efc68a"></i>工程脚本</span><span>○ 文件　◇ Go 包　实线 → 导入　虚线 · 属于包</span><span class="graph-render-count"></span></div>
    <p class="caption graph-note">全部节点保留，同目录文件聚集排列。搜索只高亮匹配节点，不裁掉网络；放大后显示文件名。Go 导入落在包节点，虚线仅表示文件属于包，不是函数调用。范围外的直接依赖标为“依赖目标”；无已解析连接的文件也保留。</p>
  </section>`;
  const $ = (selector) => container.querySelector(selector);
  const workspace = $(".graph-workspace"),
    stage = $(".graph-stage"),
    canvas = $(".graph-canvas");
  const ctx = canvas.getContext("2d"),
    mini = $(".graph-minimap canvas"),
    miniCtx = mini.getContext("2d");
  if (!ctx || !miniCtx) {
    stage.innerHTML =
      '<div class="empty">当前浏览器不支持 Canvas，请使用节点列表或文件清单。</div>';
    return { destroy() {}, setQuery() {} };
  }
  const controller = new AbortController();
  const listen = (element, event, fn, extra = {}) =>
    element.addEventListener(event, fn, {
      ...extra,
      signal: controller.signal,
    });
  let width = 1,
    height = 1,
    frame = 0,
    destroyed = false,
    hovered = "",
    selected = "",
    query = "",
    resultLimit = 40;
  let camera = options.saved?.camera ? { ...options.saved.camera } : null;
  const pointers = new Map();
  let gesture = null,
    gestureMoved = false;
  let matches = new Set(),
    neighbors = new Set(),
    selectionEdges = new Set();
  const state = { camera, selected: options.saved?.selected || "" };
  const colors = {
    src: "#8fb9ff",
    backend: "#81dec0",
    electron: "#d4a4f2",
    scripts: "#efc68a",
    docs: "#c3b9ee",
  };
  const color = (node) => colors[node.path.split("/")[0]] || "#a6bacd";
  const edgeKey = (edge) => edge.from + "\0" + edge.to + "\0" + edge.kind;
  const screen = (node) => ({
    x: node.x * camera.scale + camera.x,
    y: node.y * camera.scale + camera.y,
  });
  function remember() {
    state.camera = { ...camera };
    state.selected = selected;
    options.onState?.(state);
  }
  function schedule() {
    if (!destroyed && !frame) frame = requestAnimationFrame(draw);
  }
  function setCamera(next) {
    camera = next;
    remember();
    schedule();
  }
  function fit() {
    setCamera(fitGraph(graph.bounds, width, height));
  }
  function zoom(factor, point = { x: width / 2, y: height / 2 }) {
    setCamera(zoomCamera(camera, factor, point));
  }
  function resize() {
    const box = stage.getBoundingClientRect();
    const oldWidth = width,
      oldHeight = height;
    width = Math.max(1, box.width);
    height = Math.max(1, box.height);
    const dpr = Math.min(devicePixelRatio || 1, 2);
    canvas.width = Math.round(width * dpr);
    canvas.height = Math.round(height * dpr);
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    mini.width = 180 * dpr;
    mini.height = 116 * dpr;
    miniCtx.setTransform(dpr, 0, 0, dpr, 0, 0);
    if (!camera) camera = fitGraph(graph.bounds, width, height);
    else if (oldWidth > 1) {
      camera.x += (width - oldWidth) / 2;
      camera.y += (height - oldHeight) / 2;
    }
    remember();
    schedule();
  }
  function nodeSize(node) {
    return Math.max(1.4, node.radius * camera.scale);
  }
  function draw() {
    frame = 0;
    if (destroyed) return;
    ctx.clearRect(0, 0, width, height);
    ctx.fillStyle = "#0a111b";
    ctx.fillRect(0, 0, width, height);
    const grid = Math.max(24, 140 * camera.scale);
    ctx.fillStyle = "#253242";
    for (let x = camera.x % grid; x < width; x += grid)
      for (let y = camera.y % grid; y < height; y += grid)
        ctx.fillRect(x, y, 1, 1);
    // Directory halos provide orientation without collapsing any file nodes.
    for (const group of graph.groups) {
      const p = screen(group),
        radius = group.radius * camera.scale;
      if (
        p.x + radius < 0 ||
        p.x - radius > width ||
        p.y + radius < 0 ||
        p.y - radius > height
      )
        continue;
      ctx.beginPath();
      ctx.arc(p.x, p.y, radius, 0, Math.PI * 2);
      ctx.fillStyle = "#839fbd08";
      ctx.fill();
      ctx.strokeStyle = "#839fbd17";
      ctx.lineWidth = 1;
      ctx.stroke();
      if (radius > 40) {
        ctx.font = "11px Consolas, monospace";
        ctx.fillStyle = "#7f95ae";
        ctx.textAlign = "center";
        ctx.fillText(group.name, p.x, p.y - radius + 18);
      }
    }
    function drawEdge(edge, emphasis) {
      const from = graph.byId.get(edge.from),
        to = graph.byId.get(edge.to);
      const a = screen(from),
        b = screen(to);
      if (
        Math.max(a.x, b.x) < -20 ||
        Math.min(a.x, b.x) > width + 20 ||
        Math.max(a.y, b.y) < -20 ||
        Math.min(a.y, b.y) > height + 20
      )
        return;
      const membership = edge.kind === "membership";
      ctx.strokeStyle = emphasis
        ? membership
          ? "#edc68a88"
          : edge.from === selected
            ? "#81dec0dd"
            : "#a6c9ffdd"
        : selected
          ? "#45627c12"
          : membership
            ? "#64778b2b"
            : "#678eae35";
      ctx.lineWidth = emphasis ? 1.5 : 0.65;
      ctx.setLineDash(membership ? [3, 5] : []);
      ctx.beginPath();
      ctx.moveTo(a.x, a.y);
      ctx.lineTo(b.x, b.y);
      ctx.stroke();
      if (!membership && (emphasis || camera.scale > 1.4)) {
        const angle = Math.atan2(b.y - a.y, b.x - a.x),
          radius = nodeSize(to) + 3;
        const x = b.x - Math.cos(angle) * radius,
          y = b.y - Math.sin(angle) * radius;
        ctx.beginPath();
        ctx.moveTo(x, y);
        ctx.lineTo(
          x - 7 * Math.cos(angle - 0.4),
          y - 7 * Math.sin(angle - 0.4),
        );
        ctx.moveTo(x, y);
        ctx.lineTo(
          x - 7 * Math.cos(angle + 0.4),
          y - 7 * Math.sin(angle + 0.4),
        );
        ctx.stroke();
      }
    }
    for (const edge of graph.edges)
      if (!selectionEdges.has(edgeKey(edge))) drawEdge(edge, false);
    for (const edge of graph.edges)
      if (selectionEdges.has(edgeKey(edge))) drawEdge(edge, true);
    ctx.setLineDash([]);
    const visible = [],
      labels = [];
    for (const node of graph.nodes) {
      const p = screen(node),
        r = nodeSize(node);
      if (p.x < -100 || p.x > width + 100 || p.y < -40 || p.y > height + 40)
        continue;
      visible.push(node);
      const focus = node.id === selected || node.id === hovered,
        matched = query && matches.has(node.id),
        related = neighbors.has(node.id);
      ctx.globalAlpha =
        selected && !focus && !related
          ? 0.2
          : query && !matched && !focus
            ? 0.24
            : 1;
      ctx.fillStyle = color(node);
      ctx.beginPath();
      if (node.kind === "package") {
        ctx.moveTo(p.x, p.y - r * 1.3);
        ctx.lineTo(p.x + r, p.y);
        ctx.lineTo(p.x, p.y + r * 1.3);
        ctx.lineTo(p.x - r, p.y);
        ctx.closePath();
      } else ctx.arc(p.x, p.y, r, 0, Math.PI * 2);
      ctx.fill();
      if (focus || matched) {
        ctx.strokeStyle = focus ? "#ffffff" : "#ffe3a5";
        ctx.lineWidth = 1.8;
        ctx.beginPath();
        ctx.arc(p.x, p.y, r + 4, 0, Math.PI * 2);
        ctx.stroke();
      }
      ctx.globalAlpha = 1;
      if (focus || matched || (camera.scale > 0.85 && (!selected || related)))
        labels.push({
          node,
          p,
          r,
          priority: focus ? 3 : matched ? 2 : related ? 1 : 0,
        });
    }
    const occupied = [];
    ctx.font = '12px Consolas, "Microsoft YaHei", monospace';
    ctx.textAlign = "left";
    for (const { node, p, r, priority } of labels.sort(
      (a, b) => b.priority - a.priority,
    )) {
      const text = node.label,
        w = ctx.measureText(text).width + 8,
        x = p.x + r + 6,
        y = p.y - 8;
      if (
        !priority &&
        occupied.some(
          (rect) =>
            x < rect.x + rect.w &&
            x + w > rect.x &&
            y < rect.y + 18 &&
            y + 18 > rect.y,
        )
      )
        continue;
      occupied.push({ x, y, w });
      ctx.fillStyle = "#0a111be8";
      ctx.fillRect(x - 2, y, w, 18);
      ctx.fillStyle = priority ? "#eef5ff" : "#b4c6d9";
      ctx.fillText(text, x + 2, y + 13);
    }
    $(".graph-zoom").textContent = Math.round(camera.scale * 100) + "%";
    $(".graph-render-count").textContent =
      `视野内 ${number(visible.length)} / ${number(graph.nodes.length)} 节点`;
    canvas.dataset.scale = camera.scale.toFixed(4);
    canvas.dataset.nodes = graph.nodes.length;
    canvas.dataset.edges = graph.edges.length;
    canvas.dataset.cameraX = camera.x.toFixed(2);
    canvas.dataset.cameraY = camera.y.toFixed(2);
    drawMini();
  }
  function miniTransform() {
    const b = graph.bounds,
      scale = Math.min(164 / b.width, 100 / b.height);
    return {
      scale,
      x: (180 - b.width * scale) / 2 - b.x * scale,
      y: (116 - b.height * scale) / 2 - b.y * scale,
    };
  }
  function drawMini() {
    const t = miniTransform();
    miniCtx.clearRect(0, 0, 180, 116);
    miniCtx.fillStyle = "#0d1825";
    miniCtx.fillRect(0, 0, 180, 116);
    for (const node of graph.nodes) {
      miniCtx.fillStyle = node.id === selected ? "#fff" : color(node);
      miniCtx.fillRect(
        node.x * t.scale + t.x,
        node.y * t.scale + t.y,
        1.5,
        1.5,
      );
    }
    miniCtx.strokeStyle = "#e9f5ff";
    miniCtx.lineWidth = 1;
    const x = (-camera.x / camera.scale) * t.scale + t.x,
      y = (-camera.y / camera.scale) * t.scale + t.y;
    miniCtx.fillStyle = "#d8edff14";
    miniCtx.fillRect(
      x,
      y,
      (width / camera.scale) * t.scale,
      (height / camera.scale) * t.scale,
    );
    miniCtx.strokeRect(
      x,
      y,
      (width / camera.scale) * t.scale,
      (height / camera.scale) * t.scale,
    );
  }
  function hit(point) {
    let found = null,
      best = Infinity;
    for (const node of graph.nodes) {
      const p = screen(node),
        distance = Math.hypot(p.x - point.x, p.y - point.y);
      if (distance < Math.max(7, nodeSize(node) + 3) && distance < best) {
        best = distance;
        found = node;
      }
    }
    return found;
  }
  function point(event) {
    const rect = canvas.getBoundingClientRect();
    return { x: event.clientX - rect.left, y: event.clientY - rect.top };
  }
  function nodeButton(id) {
    const node = graph.byId.get(id);
    return `<button class="graph-node-link" data-node-id="${escape(id)}" title="${escape(node.path)}"><span>${escape(node.label)}</span><small>${escape(node.path)}</small></button>`;
  }
  function renderSelection() {
    const node = graph.byId.get(selected);
    if (!node) {
      $(".graph-selection").innerHTML =
        `<div class="eyebrow">FULL DEPENDENCY NETWORK</div><h3>每个文件，都在图上。</h3><p class="caption">从全景放大到一个节点。点击节点查看导入、被引用及包归属，也可以从下方搜索直接飞到文件。</p>`;
      return;
    }
    const list = (title, ids) =>
      ids.length
        ? `<details open><summary>${title} · ${number(ids.length)}</summary><div class="graph-neighbors">${ids.map(nodeButton).join("")}</div></details>`
        : "";
    $(".graph-selection").innerHTML =
      `<div class="graph-selected-head"><span class="badge">${node.kind === "package" ? "Go 包" : node.context ? "依赖目标" : "文件"}</span><button data-graph-action="clear">取消选择</button></div><h3 class="graph-selected-path">${escape(node.path)}</h3><p class="caption">${node.incoming.length} 条入向导入 · ${node.outgoing.length} 条出向导入${node.kind === "file" ? ` · ${number(node.lines)} 行` : ""}</p><div class="graph-selected-actions"><button data-graph-action="center">定位节点</button>${node.kind === "file" ? '<button data-graph-action="details">文件详情</button>' : ""}</div>${list("导入 →", node.outgoing)}${list("被引用 ←", node.incoming)}${list("属于包（非调用）", node.packages)}${list("包含文件（非调用）", node.members)}${node.degree || node.packages.length || node.members.length ? "" : '<p class="caption">没有已解析的本地连接；这不表示文件未被使用。</p>'}`;
  }
  function select(id, center = false) {
    selected = graph.byId.has(id) ? id : "";
    neighbors = new Set();
    selectionEdges = new Set();
    for (const edge of graph.edges)
      if (edge.from === selected || edge.to === selected) {
        neighbors.add(edge.from);
        neighbors.add(edge.to);
        selectionEdges.add(edgeKey(edge));
      }
    canvas.dataset.selected = selected;
    renderSelection();
    if (center && selected) {
      const node = graph.byId.get(selected);
      const scale = Math.max(camera.scale, 1.5);
      camera = {
        scale,
        x: width / 2 - node.x * scale,
        y: height / 2 - node.y * scale,
      };
    }
    remember();
    schedule();
  }
  function results() {
    const found = graph.nodes.filter(
      (node) =>
        !query || (node.path + " " + node.label).toLowerCase().includes(query),
    );
    matches = new Set(query ? found.map((node) => node.id) : []);
    $(".graph-search-summary").textContent = query
      ? `${number(found.length)} 个匹配 · 点击定位`
      : `${number(graph.nodes.length)} 个节点 · 按路径浏览`;
    $(".graph-results").innerHTML =
      found
        .slice(0, resultLimit)
        .map((node) => nodeButton(node.id))
        .join("") || '<p class="caption">没有匹配的路径。全图保持可见。</p>';
    $(".graph-more").hidden = found.length <= resultLimit;
    schedule();
  }
  function setQuery(value) {
    query = value.trim().toLowerCase();
    $(".graph-search").value = value;
    resultLimit = 40;
    results();
  }
  function gestureStart() {
    const values = [...pointers.values()];
    gesture =
      values.length > 1
        ? { count: 2, a: values[0], b: values[1], camera: { ...camera } }
        : { count: 1, a: values[0], camera: { ...camera } };
  }
  listen(canvas, "pointerdown", (event) => {
    if (event.button !== 0) return;
    canvas.focus();
    canvas.setPointerCapture(event.pointerId);
    pointers.set(event.pointerId, point(event));
    gestureMoved = pointers.size > 1;
    gestureStart();
    $(".graph-hover").hidden = true;
  });
  listen(canvas, "pointermove", (event) => {
    const p = point(event);
    if (!pointers.has(event.pointerId)) {
      const node = hit(p);
      if ((node?.id || "") !== hovered) {
        hovered = node?.id || "";
        schedule();
      }
      canvas.style.cursor = node ? "pointer" : "grab";
      const tooltip = $(".graph-hover");
      tooltip.hidden = !node;
      if (node) {
        tooltip.textContent = node.path;
        tooltip.style.left =
          Math.max(8, Math.min(width - 250, p.x + 14)) + "px";
        tooltip.style.top = Math.max(8, Math.min(height - 60, p.y + 14)) + "px";
      }
      return;
    }
    pointers.set(event.pointerId, p);
    const values = [...pointers.values()];
    if (gesture.count === 2 && values.length > 1) {
      const initial = {
        x: (gesture.a.x + gesture.b.x) / 2,
        y: (gesture.a.y + gesture.b.y) / 2,
      };
      const current = {
        x: (values[0].x + values[1].x) / 2,
        y: (values[0].y + values[1].y) / 2,
      };
      const factor =
        Math.hypot(values[0].x - values[1].x, values[0].y - values[1].y) /
        Math.max(
          1,
          Math.hypot(gesture.a.x - gesture.b.x, gesture.a.y - gesture.b.y),
        );
      const next = zoomCamera(gesture.camera, factor, initial);
      next.x += current.x - initial.x;
      next.y += current.y - initial.y;
      setCamera(next);
      gestureMoved = true;
    } else {
      const dx = p.x - gesture.a.x,
        dy = p.y - gesture.a.y;
      if (Math.hypot(dx, dy) > 4) gestureMoved = true;
      setCamera({
        ...gesture.camera,
        x: gesture.camera.x + dx,
        y: gesture.camera.y + dy,
      });
    }
  });
  function pointerEnd(event) {
    if (!pointers.has(event.pointerId)) return;
    const click =
      !gestureMoved && pointers.size === 1 && event.type === "pointerup";
    pointers.delete(event.pointerId);
    if (canvas.hasPointerCapture(event.pointerId))
      canvas.releasePointerCapture(event.pointerId);
    if (click) select(hit(point(event))?.id || "");
    if (pointers.size) {
      gestureStart();
      gestureMoved = true;
    } else gesture = null;
  }
  listen(canvas, "pointerup", pointerEnd);
  listen(canvas, "pointercancel", pointerEnd);
  listen(canvas, "pointerleave", () => {
    hovered = "";
    $(".graph-hover").hidden = true;
    schedule();
  });
  listen(
    canvas,
    "wheel",
    (event) => {
      event.preventDefault();
      const delta =
        event.deltaY *
        (event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? height : 1);
      zoom(
        Math.exp(-Math.max(-300, Math.min(300, delta)) * 0.002),
        point(event),
      );
    },
    { passive: false },
  );
  listen(canvas, "dblclick", (event) => {
    const node = hit(point(event));
    if (node) select(node.id, true);
    else zoom(1.8, point(event));
  });
  listen(canvas, "keydown", (event) => {
    if (
      [
        "+",
        "=",
        "-",
        "0",
        "ArrowLeft",
        "ArrowRight",
        "ArrowUp",
        "ArrowDown",
        "Escape",
      ].includes(event.key)
    )
      event.preventDefault();
    if (event.key === "+" || event.key === "=") zoom(1.3);
    else if (event.key === "-") zoom(1 / 1.3);
    else if (event.key === "0") fit();
    else if (event.key === "Escape") {
      if (workspace.classList.contains("graph-expanded")) toggleFullscreen();
      else select("");
    } else if (event.key.startsWith("Arrow"))
      setCamera({
        ...camera,
        x:
          camera.x +
          (event.key === "ArrowLeft"
            ? 70
            : event.key === "ArrowRight"
              ? -70
              : 0),
        y:
          camera.y +
          (event.key === "ArrowUp" ? 70 : event.key === "ArrowDown" ? -70 : 0),
      });
  });
  listen(mini, "pointerdown", (event) => {
    const rect = mini.getBoundingClientRect(),
      t = miniTransform(),
      x = ((event.clientX - rect.left) * 180) / rect.width,
      y = ((event.clientY - rect.top) * 116) / rect.height;
    setCamera({
      ...camera,
      x: width / 2 - ((x - t.x) / t.scale) * camera.scale,
      y: height / 2 - ((y - t.y) / t.scale) * camera.scale,
    });
  });
  let previousOverflow = "";
  function toggleFullscreen() {
    const expanding = !workspace.classList.contains("graph-expanded");
    workspace.classList.toggle("graph-expanded", expanding);
    if (expanding) {
      previousOverflow = document.body.style.overflow;
      document.body.style.overflow = "hidden";
    } else document.body.style.overflow = previousOverflow;
    $('[data-graph-action="fullscreen"]').textContent = expanding
      ? "退出展开（Esc）"
      : "展开画布";
    canvas.focus();
  }
  listen(workspace, "keydown", (event) => {
    if (
      event.key === "Escape" &&
      workspace.classList.contains("graph-expanded") &&
      event.target !== canvas
    ) {
      toggleFullscreen();
      event.stopPropagation();
    }
  });
  listen(workspace, "click", (event) => {
    const node = event.target.closest("[data-node-id]");
    if (node) return select(node.dataset.nodeId, true);
    const action = event.target.closest("[data-graph-action]")?.dataset
      .graphAction;
    if (action === "in") zoom(1.4);
    else if (action === "out") zoom(1 / 1.4);
    else if (action === "fit") fit();
    else if (action === "clear") select("");
    else if (action === "center") select(selected, true);
    else if (action === "fullscreen") toggleFullscreen();
    else if (action === "details") options.onFile?.(selected);
  });
  listen($(".graph-search"), "input", (event) => {
    query = event.target.value.trim().toLowerCase();
    resultLimit = 40;
    results();
  });
  listen($(".graph-search"), "keydown", (event) => {
    if (event.key === "Enter") {
      const first = graph.nodes.find((node) =>
        (node.path + " " + node.label).toLowerCase().includes(query),
      );
      if (first) select(first.id, true);
    }
  });
  listen($(".graph-more"), "click", () => {
    resultLimit += 60;
    results();
  });
  const observer = new ResizeObserver(resize);
  observer.observe(stage);
  const initialSelected = state.selected;
  resize();
  select(initialSelected);
  setQuery(options.query || "");
  return {
    setQuery,
    state,
    destroy() {
      destroyed = true;
      controller.abort();
      observer.disconnect();
      cancelAnimationFrame(frame);
      if (workspace.classList.contains("graph-expanded"))
        document.body.style.overflow = previousOverflow;
    },
  };
}
