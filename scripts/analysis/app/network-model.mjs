// Pure graph semantics and deterministic spatial layout, shared with offline HTML.
export function buildFileGraph(data, files) {
  const all = new Map(data.files.map((file) => [file.path, file]));
  const nodes = new Map();
  const edges = [];
  const seen = new Set();
  function addFile(file, context = false) {
    if (!nodes.has(file.path))
      nodes.set(file.path, {
        id: file.path,
        path: file.path,
        label: file.path.split("/").at(-1),
        module: file.module,
        kind: "file",
        context,
        lines: file.lines,
        incoming: [],
        outgoing: [],
        members: [],
        packages: [],
      });
    return nodes.get(file.path);
  }
  function addPackage(path) {
    const id = "package:" + path;
    if (!nodes.has(id))
      nodes.set(id, {
        id,
        path,
        label: path.split("/").at(-1) + " [Go 包]",
        module: path.split("/").slice(0, 3).join("/"),
        kind: "package",
        context: false,
        lines: 0,
        incoming: [],
        outgoing: [],
        members: [],
        packages: [],
      });
    return nodes.get(id);
  }
  function connect(from, to, kind) {
    const key = from.id + "\0" + to.id + "\0" + kind;
    if (seen.has(key)) return;
    seen.add(key);
    const edge = { from: from.id, to: to.id, kind };
    edges.push(edge);
    if (kind === "import") {
      from.outgoing.push(to.id);
      to.incoming.push(from.id);
    } else {
      from.packages.push(to.id);
      to.members.push(from.id);
    }
  }
  const paths = new Set(files.map((file) => file.path));
  for (const file of files) addFile(file);
  for (const edge of data.edges) {
    if (!paths.has(edge.from)) continue;
    const from = nodes.get(edge.from);
    const to =
      edge.type === "package"
        ? addPackage(edge.to)
        : all.has(edge.to)
          ? addFile(all.get(edge.to), !paths.has(edge.to))
          : null;
    if (to) connect(from, to, "import");
  }
  for (const file of files) {
    if (file.ext === ".go")
      connect(
        nodes.get(file.path),
        addPackage(file.path.slice(0, file.path.lastIndexOf("/"))),
        "membership",
      );
  }
  const list = [...nodes.values()].sort((a, b) => a.id.localeCompare(b.id));
  for (const node of list) {
    node.degree = node.incoming.length + node.outgoing.length;
    node.radius =
      node.kind === "package"
        ? 10
        : Math.min(12, 4 + Math.log2(1 + node.degree) * 1.1);
  }
  const groups = layoutFileGraph(list);
  return { nodes: list, byId: nodes, edges, groups, bounds: graphBounds(list) };
}

export function layoutFileGraph(nodes) {
  const map = new Map();
  for (const node of nodes) {
    if (!map.has(node.module)) map.set(node.module, []);
    map.get(node.module).push(node);
  }
  const groups = [...map]
    .map(([name, items]) => ({
      name,
      nodes: items,
      radius: Math.max(85, Math.sqrt(items.length) * 30 + 40),
      x: 0,
      y: 0,
    }))
    .sort((a, b) => b.radius - a.radius || a.name.localeCompare(b.name));
  // Pack non-overlapping directory clusters around the center. Stable coordinates
  // keep navigation reproducible; node positions do not drift while inspecting.
  const placed = [];
  for (const group of groups) {
    if (placed.length) {
      for (let step = 1; ; step++) {
        const angle = step * 2.399963229728653;
        const distance = Math.sqrt(step) * 36;
        group.x = Math.cos(angle) * distance;
        group.y = Math.sin(angle) * distance;
        if (
          placed.every(
            (other) =>
              Math.hypot(group.x - other.x, group.y - other.y) >=
              group.radius + other.radius + 45,
          )
        )
          break;
      }
    }
    const ordered = [...group.nodes].sort(
      (a, b) => b.degree - a.degree || a.id.localeCompare(b.id),
    );
    ordered.forEach((node, index) => {
      const angle = index * 2.399963229728653;
      const radius = Math.sqrt(index) * 29;
      node.x = group.x + Math.cos(angle) * radius;
      node.y = group.y + Math.sin(angle) * radius;
    });
    placed.push(group);
  }
  return groups;
}

export function graphBounds(nodes) {
  if (!nodes.length) return { x: -100, y: -100, width: 200, height: 200 };
  const xs = nodes.map((node) => node.x),
    ys = nodes.map((node) => node.y);
  const x = Math.min(...xs) - 90,
    y = Math.min(...ys) - 90;
  return {
    x,
    y,
    width: Math.max(...xs) - x + 90,
    height: Math.max(...ys) - y + 90,
  };
}

export function zoomCamera(camera, factor, point, min = 0.025, max = 8) {
  const scale = Math.min(max, Math.max(min, camera.scale * factor));
  const ratio = scale / camera.scale;
  return {
    scale,
    x: point.x - (point.x - camera.x) * ratio,
    y: point.y - (point.y - camera.y) * ratio,
  };
}

export function fitGraph(bounds, width, height) {
  const scale = Math.min(
    2,
    Math.max(
      0.025,
      Math.min((width - 90) / bounds.width, (height - 90) / bounds.height),
    ),
  );
  return {
    scale,
    x: width / 2 - (bounds.x + bounds.width / 2) * scale,
    y: height / 2 - (bounds.y + bounds.height / 2) * scale,
  };
}
