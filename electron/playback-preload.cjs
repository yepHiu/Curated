const { contextBridge, ipcRenderer } = require("electron")
// 本地控件只操作已有 session；没有连接、PIN、检索、文件选择或任意命令能力。
contextBridge.exposeInMainWorld("curatedPlayer", {
  snapshot: () => ipcRenderer.invoke("curated:playback-snapshot"),
  command: (sessionId, input) => ipcRenderer.invoke("curated:playback-command", sessionId, input),
  capture: (sessionId, retryId) => ipcRenderer.invoke("curated:playback-capture", sessionId, retryId),
  capturePreferences: () => ipcRenderer.invoke("curated:playback-capture-preferences"),
  subscribe: callback => {
    const listener = (_event, state) => callback(state)
    ipcRenderer.on("curated:playback-state", listener)
    return () => ipcRenderer.removeListener("curated:playback-state", listener)
  },
})
