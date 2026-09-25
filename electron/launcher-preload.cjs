const { contextBridge, ipcRenderer } = require("electron")
contextBridge.exposeInMainWorld("curatedConnection", {
  platform: process.platform,
  checkUpdate: () => ipcRenderer.invoke("curated:desktop-update"),
  discover: () => ipcRenderer.invoke("curated:discover"),
  list: () => ipcRenderer.invoke("curated:connections"),
  connect: (url) => ipcRenderer.invoke("curated:connect", url),
  cancel: () => ipcRenderer.invoke("curated:cancel-connect"),
  forget: (url) => ipcRenderer.invoke("curated:forget", url),
})
