const { contextBridge, ipcRenderer } = require("electron")

contextBridge.exposeInMainWorld("javLibrary", {
  windowChrome: process.platform === "darwin" ? "macos" : "native",
  pickDirectory: () => ipcRenderer.invoke("curated:pick-directory"),
})
