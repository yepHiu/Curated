const { contextBridge, ipcRenderer } = require("electron")
// Server paths must never be filled using the client's native folder picker.
contextBridge.exposeInMainWorld("curatedDesktop", {
  bridgeVersion: 1,
  serverPaths: true,
  getInfo: () => ipcRenderer.invoke("curated:desktop-info"),
  changeServer: () => ipcRenderer.invoke("curated:change-server"),
})
