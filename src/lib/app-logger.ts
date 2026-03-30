import log from "loglevel"

/** localStorage key for settings UI「前端日志级别」 */
export const CLIENT_LOG_LEVEL_KEY = "curated-client-log-level"

export const CLIENT_LOG_LEVEL_OPTIONS = [
  "trace",
  "debug",
  "info",
  "warn",
  "error",
  "silent",
] as const

export type ClientLogLevelName = (typeof CLIENT_LOG_LEVEL_OPTIONS)[number]

function isClientLogLevelName(s: string): s is ClientLogLevelName {
  return (CLIENT_LOG_LEVEL_OPTIONS as readonly string[]).includes(s)
}

function normalizeLevelInput(s: string | null | undefined): ClientLogLevelName | null {
  const v = (s ?? "").trim().toLowerCase()
  return isClientLogLevelName(v) ? v : null
}

/**
 * 优先级：localStorage > VITE_LOG_LEVEL > 开发 debug / 生产 warn。
 * 在应用入口尽早调用一次。
 */
export function initClientLogger(): void {
  const defaultLevel: ClientLogLevelName = import.meta.env.DEV ? "debug" : "warn"
  let level: ClientLogLevelName = defaultLevel
  try {
    const stored = normalizeLevelInput(localStorage.getItem(CLIENT_LOG_LEVEL_KEY))
    if (stored) {
      level = stored
    } else {
      const fromEnv = normalizeLevelInput(import.meta.env.VITE_LOG_LEVEL as string | undefined)
      if (fromEnv) level = fromEnv
    }
  } catch {
    // private mode / no localStorage
    const fromEnv = normalizeLevelInput(import.meta.env.VITE_LOG_LEVEL as string | undefined)
    if (fromEnv) level = fromEnv
  }
  log.setLevel(level)
}

/** 当前生效的前端日志级别名称（与 loglevel 内部数字对应） */
export function getClientLogLevelName(): ClientLogLevelName {
  const n = log.getLevel()
  const map: Record<number, ClientLogLevelName> = {
    [log.levels.TRACE]: "trace",
    [log.levels.DEBUG]: "debug",
    [log.levels.INFO]: "info",
    [log.levels.WARN]: "warn",
    [log.levels.ERROR]: "error",
    [log.levels.SILENT]: "silent",
  }
  return map[n] ?? "info"
}

/** 设置级别并持久化到 localStorage（若可用） */
export function setClientLogLevel(name: ClientLogLevelName): void {
  log.setLevel(name)
  try {
    localStorage.setItem(CLIENT_LOG_LEVEL_KEY, name)
  } catch {
    // ignore
  }
}

/** 默认 loglevel 根 logger，供业务模块使用 */
export const appLogger = log
