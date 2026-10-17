export interface AtlasFile {
  path: string;
  ext: string;
  kind: "source" | "test" | "docs" | "config" | "asset";
  module: string;
  bytes: number;
  lines: number;
  nonblank: number;
  functions: number;
  maxBranches: number;
  parsed: boolean;
}
export interface AtlasFunction {
  name: string;
  path: string;
  kind: AtlasFile["kind"];
  module: string;
  line: number;
  end: number;
  lines: number;
  branches: number;
}
export interface AtlasSnapshot {
  schema: number;
  generatedAt: string;
  commit: string;
  branch: string;
  dirty: boolean;
  files: AtlasFile[];
  functions: AtlasFunction[];
  edges: {
    from: string;
    to: string;
    type: "file" | "package";
    source: string;
    target: string;
  }[];
  warnings: { path: string; message: string }[];
  unresolved: { from: string; spec: string }[];
  excluded: string[];
  counts: {
    files: number;
    source: number;
    sourceLines: number;
    functions: number;
    parsedFiles: number;
  };
}
export interface ScanStatus {
  sessionId: string;
  scanning: boolean;
  revision: number;
  phase: string;
  message: string;
  startedAt: string | null;
  finishedAt: string | null;
  error: string | null;
  hasSnapshot: boolean;
  completed?: number;
  total?: number;
}
export interface ExplorerState {
  view: string;
  scope: string;
  query: string;
  queryInput: string;
  dir: string;
  metric: string;
  sort: string;
  page: number;
  module: string;
}
