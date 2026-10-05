const { contextBridge, ipcRenderer } = require("electron")

// 所有函数只承载受限原型动作，不暴露 ipcRenderer、Cookie 或通用命令执行。
contextBridge.exposeInMainWorld("nativePlayerLab", {
  /** 获取状态快照。 */
  getStatus: () => ipcRenderer.invoke("native-lab:status"),
  /** 主进程弹出 mpv.exe 文件选择器。 */
  selectExecutable: () => ipcRenderer.invoke("native-lab:engine"),
  /** 验证并连接指定 Server 根地址。 */
  connect: (origin) => ipcRenderer.invoke("native-lab:connect", origin),
  /** 对当前 Server 解锁。 */
  unlock: (pin) => ipcRenderer.invoke("native-lab:unlock", pin),
  /** 检索最多三十部影片。 */
  search: (query) => ipcRenderer.invoke("native-lab:search", query),
  /** 读取文件名与稳定文件身份。 */
  detail: (movieId) => ipcRenderer.invoke("native-lab:detail", movieId),
  /** 读取逐文件续播起点。 */
  resume: (input) => ipcRenderer.invoke("native-lab:resume", input),
  /** 启动原始流原生播放。 */
  start: (input) => ipcRenderer.invoke("native-lab:start", input),
  /** 执行枚举控制。 */
  control: (input) => ipcRenderer.invoke("native-lab:control", input),
  /** 只暴露窗口动作枚举，不暴露句柄、位置或原生通用调用。 */
  windowAction: (action) => ipcRenderer.invoke("native-lab:window", action),
})
