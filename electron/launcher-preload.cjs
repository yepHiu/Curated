const { contextBridge, ipcRenderer } = require("electron")
contextBridge.exposeInMainWorld("curatedConnection", {
  platform: process.platform,
  // 设置通道仅暴露给随包本地连接页。
  readSettings: () => ipcRenderer.invoke("curated:settings-read"),
  // 主进程重新校验所有输入，不信任 renderer 的表单约束。
  saveSettings: (value) => ipcRenderer.invoke("curated:settings-save", value),
  checkUpdate: () => ipcRenderer.invoke("curated:desktop-update"),
  discover: () => ipcRenderer.invoke("curated:discover"),
  onError: (callback) => {
    const listener = (_event, message) => callback(message)
    ipcRenderer.on("curated:connection-error", listener)
    return () => ipcRenderer.removeListener("curated:connection-error", listener)
  },
  list: () => ipcRenderer.invoke("curated:connections"),
  connect: (url) => ipcRenderer.invoke("curated:connect", url),
  cancel: () => ipcRenderer.invoke("curated:cancel-connect"),
  forget: (url) => ipcRenderer.invoke("curated:forget", url),
})
