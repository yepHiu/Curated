const { contextBridge, ipcRenderer } = require("electron")

contextBridge.exposeInMainWorld("javLibrary", {
  openServerConnections: (serverId) => ipcRenderer.invoke("curated:open-servers", serverId),
  getServerConnections: () => ipcRenderer.invoke("curated:server-connections"),
  windowChrome: process.platform === "darwin" ? "macos" : "native",
  getDesktopInfo: () => ipcRenderer.invoke("curated:desktop-info"),
  checkDesktopUpdate: () => ipcRenderer.invoke("curated:desktop-check-update"),
  pickDirectory: () => ipcRenderer.invoke("curated:pick-directory"),
})
