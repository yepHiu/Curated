const { contextBridge, ipcRenderer } = require("electron")
contextBridge.exposeInMainWorld("curatedConnection", {
  list: () => ipcRenderer.invoke("curated:connections"),
  connect: (url) => ipcRenderer.invoke("curated:connect", url),
  cancel: () => ipcRenderer.invoke("curated:cancel-connect"),
  forget: (url) => ipcRenderer.invoke("curated:forget", url),
})
