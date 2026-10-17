import type { AtlasSnapshot, ExplorerState } from "./types";
export function mountAtlas(
  host: ShadowRoot | HTMLElement,
  data: AtlasSnapshot,
  previousState?: Partial<ExplorerState>,
  live?: boolean,
): { state: ExplorerState; destroy(): void };
