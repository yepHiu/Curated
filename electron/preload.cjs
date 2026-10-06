const { contextBridge, ipcRenderer } = require("electron")

contextBridge.exposeInMainWorld("javLibrary", {
  playback: {
    capabilities: () => ipcRenderer.invoke("curated:playback-capabilities"),
    setPreferences: input => ipcRenderer.invoke("curated:playback-preferences", input),
    open: input => ipcRenderer.invoke("curated:playback-open", input),
    snapshot: () => ipcRenderer.invoke("curated:playback-snapshot"),
    command: (sessionId, input) => ipcRenderer.invoke("curated:playback-command", sessionId, input),
    subscribe: callback => {
      const listener = (_event, snapshot) => callback(snapshot)
      ipcRenderer.on("curated:playback-state", listener)
      return () => ipcRenderer.removeListener("curated:playback-state", listener)
    },
    onWebFallback: callback => {
      const listener = (_event, input) => callback(input)
      ipcRenderer.on("curated:playback-web", listener)
      return () => ipcRenderer.removeListener("curated:playback-web", listener)
    },
  },
  openServerConnections: (serverId) => ipcRenderer.invoke("curated:open-servers", serverId),
  addServer: (input) => ipcRenderer.invoke("curated:add-server", input),
  getServerConnections: () => ipcRenderer.invoke("curated:server-connections"),
  windowChrome: process.platform === "darwin" ? "macos" : "native",
  getDesktopInfo: () => ipcRenderer.invoke("curated:desktop-info"),
  checkDesktopUpdate: () => ipcRenderer.invoke("curated:desktop-check-update"),
  pickDirectory: () => ipcRenderer.invoke("curated:pick-directory"),
})
