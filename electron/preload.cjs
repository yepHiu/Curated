const { contextBridge, ipcRenderer } = require("electron")

contextBridge.exposeInMainWorld("javLibrary", {
  windowChrome: process.platform === "darwin" ? "macos" : "native",
  getDesktopInfo: () => ipcRenderer.invoke("curated:desktop-info"),
  checkDesktopUpdate: () => ipcRenderer.invoke("curated:desktop-check-update"),
  pickDirectory: () => ipcRenderer.invoke("curated:pick-directory"),
})
