const { contextBridge, ipcRenderer } = require("electron")
contextBridge.exposeInMainWorld("curatedConnections", {
  checkUpdate: () => ipcRenderer.invoke("curated:connections", "check-update"),
  downloadUpdate: () => ipcRenderer.invoke("curated:connections", "download-update"),
  list: () => ipcRenderer.invoke("curated:connections", "list"),
  save: value => ipcRenderer.invoke("curated:connections", "save", value),
  remove: id => ipcRenderer.invoke("curated:connections", "remove", id),
  connect: id => ipcRenderer.invoke("curated:connections", "connect", id),
})
