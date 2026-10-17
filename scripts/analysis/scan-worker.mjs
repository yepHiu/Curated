import { parentPort } from "node:worker_threads";
import { scanRepository } from "./code-atlas.mjs";

try {
  const data = scanRepository({
    onProgress: (progress) =>
      parentPort.postMessage({ type: "progress", progress }),
  });
  parentPort.postMessage({ type: "result", data });
} catch (error) {
  parentPort.postMessage({
    type: "error",
    message: error instanceof Error ? error.message : String(error),
  });
}
