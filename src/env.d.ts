/// <reference types="vite/client" />

/** 未来 Electron / 桌面壳注入：返回本机绝对路径 */
interface Window {
  javLibrary?: {
    pickDirectory?: () => Promise<{ path: string } | null | undefined>
  }
}

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string
  readonly VITE_USE_WEB_API?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
