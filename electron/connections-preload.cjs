const { contextBridge, ipcRenderer } = require("electron")
contextBridge.exposeInMainWorld("curatedConnections", {
  list: () => ipcRenderer.invoke("curated:connections", "list"),
  save: value => ipcRenderer.invoke("curated:connections", "save", value),
  remove: id => ipcRenderer.invoke("curated:connections", "remove", id),
  connect: id => ipcRenderer.invoke("curated:connections", "connect", id),
})
