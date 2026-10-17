import { Worker } from "node:worker_threads";
import { randomUUID } from "node:crypto";

export function runWorker(onProgress, signal) {
  return new Promise((resolve, reject) => {
    if (signal.aborted) return reject(new Error("扫描已取消"));
    const worker = new Worker(new URL("./scan-worker.mjs", import.meta.url));
    let settled = false;
    const finish = (error, data) => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      signal.removeEventListener("abort", abort);
      void worker.terminate();
      if (error) reject(error);
      else resolve(data);
    };
    const abort = () => finish(new Error("扫描已取消"));
    const timer = setTimeout(
      () => finish(new Error("扫描超过 120 秒，请检查 Git、Go 环境后重试")),
      120_000,
    );
    signal.addEventListener("abort", abort, { once: true });
    worker.on("message", (message) => {
      if (message.type === "progress") onProgress(message.progress);
      else if (message.type === "result") finish(null, message.data);
      else if (message.type === "error") finish(new Error(message.message));
    });
    worker.on("error", (error) => finish(error));
    worker.on("exit", (code) => {
      if (!settled) finish(new Error(`扫描进程意外退出 (${code})`));
    });
  });
}

export function createScanStore(scanner = runWorker) {
  let snapshot = null,
    active = null,
    disposed = false;
  const controller = new AbortController();
  const state = {
    sessionId: randomUUID(),
    scanning: false,
    revision: 0,
    phase: "idle",
    message: "等待首次扫描",
    startedAt: null,
    finishedAt: null,
    error: null,
  };
  const status = () => ({ ...state, hasSnapshot: snapshot !== null });
  function scan() {
    if (disposed) return Promise.reject(new Error("分析服务已关闭"));
    if (active) return active;
    Object.assign(state, {
      scanning: true,
      phase: "starting",
      message: "准备扫描",
      completed: undefined,
      total: undefined,
      startedAt: new Date().toISOString(),
      error: null,
    });
    active = Promise.resolve()
      .then(() =>
        scanner((progress) => {
          if (!disposed)
            Object.assign(
              state,
              { completed: undefined, total: undefined },
              progress,
            );
        }, controller.signal),
      )
      .then((data) => {
        if (disposed) return;
        snapshot = data;
        state.revision++;
        Object.assign(state, {
          phase: "ready",
          message: "扫描完成",
          finishedAt: new Date().toISOString(),
        });
      })
      .catch((error) => {
        if (!disposed)
          Object.assign(state, {
            phase: "error",
            message: "扫描失败，保留上次成功结果",
            error: error instanceof Error ? error.message : String(error),
          });
      })
      .finally(() => {
        state.scanning = false;
        active = null;
      });
    return active;
  }
  return {
    scan,
    status,
    snapshot: () => snapshot,
    dispose() {
      disposed = true;
      controller.abort();
    },
  };
}
