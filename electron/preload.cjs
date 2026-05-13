const { contextBridge, ipcRenderer } = require("electron")

contextBridge.exposeInMainWorld("javLibrary", {
  pickDirectory: () => ipcRenderer.invoke("curated:pick-directory"),
})
