import { execFile, spawn, type ChildProcessByStdio } from "node:child_process"
import type { Readable } from "node:stream"
import { setTimeout as delay } from "node:timers/promises"

type ElectronEnv = Partial<Record<string, string | undefined>>

export interface FrontendLaunchPlan {
  command: string
  args: string[]
  cwd: string
  env: NodeJS.ProcessEnv
}

type FrontendChildProcess = ChildProcessByStdio<null, Readable, Readable>

interface FrontendLaunchPlanOptions {
  appPath: string
  baseUrl?: string
  backendBaseUrl?: string
  env?: ElectronEnv
  isWindows?: boolean
}

interface StartFrontendOptions extends FrontendLaunchPlanOptions {
  fetchImpl?: typeof fetch
  timeoutMs?: number
}

export interface ManagedFrontend {
  baseUrl: string
  attachedToExisting: boolean
  process?: FrontendChildProcess
  stop: () => Promise<void>
}

const frontendUrlEnvKey = "CURATED_ELECTRON_FRONTEND_URL"
const defaultFrontendPort = "5173"

export function defaultFrontendBaseUrl(): string {
  return `http://127.0.0.1:${defaultFrontendPort}`
}

export function shouldStartDevFrontend(options: { isPackaged: boolean }): boolean {
  return !options.isPackaged
}

export function resolveFrontendBaseUrl(env: ElectronEnv = process.env): string {
  const value = env[frontendUrlEnvKey]?.trim()
  return normalizeFrontendBaseUrl(value || defaultFrontendBaseUrl())
}

/** A standalone development Desktop uses the running Vite UI for the local dev Server. */
export async function resolveRendererBaseUrl(options: {
  backendBaseUrl: string
  isPackaged: boolean
  env?: ElectronEnv
  fetchImpl?: typeof fetch
}): Promise<string> {
  const { backendBaseUrl, isPackaged } = options
  if (isPackaged) return backendBaseUrl
  const backend = new URL(backendBaseUrl)
  if (backend.protocol !== "http:" || !["127.0.0.1", "localhost"].includes(backend.hostname) || backend.port !== "8080") return backendBaseUrl

  const fetchImpl = options.fetchImpl ?? fetch
  let health: Response
  try {
    health = await fetchImpl(`${backendBaseUrl}/api/health`, {
      signal: AbortSignal.timeout(1500), redirect: "error", credentials: "omit", cache: "no-store",
    })
    if (!health.ok || (await health.json() as { name?: unknown }).name !== "curated-dev") return backendBaseUrl
  } catch { return backendBaseUrl }

  const frontendUrl = resolveFrontendBaseUrl(options.env)
  try {
    const response = await fetchImpl(frontendUrl, {
      signal: AbortSignal.timeout(1500), redirect: "error", credentials: "omit", cache: "no-store",
    })
    if (response.ok && response.headers.get("content-type")?.includes("text/html") &&
        /src="\/src\/main\.ts(?:\?[^\"]*)?"/.test(await response.text())) return frontendUrl
  } catch { /* The local Vite server is not running. */ }
  throw new Error(`本机开发 Server 已连接，但开发前端未在 ${frontendUrl} 就绪。请先运行 pnpm dev，再重试连接。`)
}

export function resolveFrontendLaunchPlan(options: FrontendLaunchPlanOptions): FrontendLaunchPlan {
  const baseUrl = options.baseUrl ?? resolveFrontendBaseUrl(options.env)
  const parsed = new URL(baseUrl)
  const isWindows = options.isWindows ?? process.platform === "win32"
  const viteArgs = ["exec", "vite", "--host", resolveViteHost(parsed.hostname), "--port", parsed.port || defaultFrontendPort]

  return {
    command: isWindows ? "cmd.exe" : "pnpm",
    args: isWindows ? ["/d", "/s", "/c", "pnpm.cmd", ...viteArgs] : viteArgs,
    cwd: normalizeFilesystemPath(options.appPath),
    env: resolveFrontendEnv(options.env, options.backendBaseUrl),
  }
}

export async function startFrontend(options: StartFrontendOptions): Promise<ManagedFrontend> {
  const fetchImpl = options.fetchImpl ?? fetch
  const baseUrl = options.baseUrl ?? resolveFrontendBaseUrl(options.env)

  if (await isFrontendHealthy(baseUrl, fetchImpl)) {
    return {
      baseUrl,
      attachedToExisting: true,
      stop: async () => {},
    }
  }

  const plan = resolveFrontendLaunchPlan({ ...options, baseUrl })
  const frontendProcess = spawn(plan.command, plan.args, {
    cwd: plan.cwd,
    env: plan.env,
    stdio: ["ignore", "pipe", "pipe"],
    windowsHide: true,
  })

  frontendProcess.stdout.on("data", (chunk: Buffer) => {
    process.stdout.write(`[curated-frontend] ${chunk.toString()}`)
  })
  frontendProcess.stderr.on("data", (chunk: Buffer) => {
    process.stderr.write(`[curated-frontend] ${chunk.toString()}`)
  })

  try {
    await waitForFrontendHealth(baseUrl, fetchImpl, options.timeoutMs ?? 20_000)
  } catch (error) {
    await stopFrontendProcess(frontendProcess)
    throw error
  }

  return {
    baseUrl,
    attachedToExisting: false,
    process: frontendProcess,
    stop: () => stopFrontendProcess(frontendProcess),
  }
}

export function shouldStopFrontendOnQuit(options: { attachedToExistingFrontend: boolean }): boolean {
  return !options.attachedToExistingFrontend
}

async function waitForFrontendHealth(baseUrl: string, fetchImpl: typeof fetch, timeoutMs: number): Promise<void> {
  const deadline = Date.now() + timeoutMs
  let lastError: unknown
  while (Date.now() < deadline) {
    try {
      if (await isFrontendHealthy(baseUrl, fetchImpl)) {
        return
      }
    } catch (error) {
      lastError = error
    }
    await delay(250)
  }
  throw new Error(`Curated frontend did not become ready at ${baseUrl}: ${formatUnknownError(lastError)}`)
}

async function isFrontendHealthy(baseUrl: string, fetchImpl: typeof fetch): Promise<boolean> {
  try {
    const response = await fetchImpl(baseUrl, { signal: AbortSignal.timeout(1000) })
    if (!response.ok) {
      return false
    }
    const contentType = response.headers.get("content-type")?.toLowerCase() ?? ""
    return contentType.includes("text/html")
  } catch {
    return false
  }
}

async function stopFrontendProcess(frontendProcess: FrontendChildProcess): Promise<void> {
  if (frontendProcess.exitCode !== null || frontendProcess.killed) {
    return
  }

  await new Promise<void>((resolve) => {
    const timeout = setTimeout(() => {
      if (frontendProcess.exitCode === null && !frontendProcess.killed) {
        frontendProcess.kill("SIGKILL")
      }
      resolve()
    }, 3000)

    frontendProcess.once("exit", () => {
      clearTimeout(timeout)
      resolve()
    })

    if (process.platform === "win32" && frontendProcess.pid) {
      execFile("taskkill", ["/pid", String(frontendProcess.pid), "/t", "/f"], { windowsHide: true }, () => {})
      return
    }

    frontendProcess.kill()
  })
}

function resolveFrontendEnv(env: ElectronEnv = process.env, backendBaseUrl?: string): NodeJS.ProcessEnv {
  const resolved: NodeJS.ProcessEnv = {
    ...process.env,
    ...env,
    VITE_USE_WEB_API: "true",
    BROWSER: "none",
  }

  if (!resolved.VITE_API_BASE_URL && backendBaseUrl) {
    resolved.VITE_API_BASE_URL = `${normalizeHttpOrigin(backendBaseUrl)}/api`
  }

  return resolved
}

function normalizeFrontendBaseUrl(rawUrl: string): string {
  const parsed = new URL(normalizeHttpOrigin(rawUrl))
  if (!parsed.port && parsed.protocol === "http:" && isLoopbackHost(parsed.hostname)) {
    parsed.port = defaultFrontendPort
  }
  return parsed.origin
}

function normalizeHttpOrigin(rawUrl: string): string {
  const parsed = new URL(rawUrl)
  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new Error(`Unsupported Curated URL protocol: ${parsed.protocol}`)
  }
  if (isLocalhostName(parsed.hostname)) {
    parsed.hostname = "127.0.0.1"
  }
  return parsed.origin
}

function resolveViteHost(hostname: string): string {
  return isLocalhostName(hostname) ? "127.0.0.1" : stripIpv6Brackets(hostname)
}

function isLocalhostName(hostname: string): boolean {
  const normalized = stripIpv6Brackets(hostname).toLowerCase()
  return normalized === "localhost" || normalized === "::1"
}

function isLoopbackHost(hostname: string): boolean {
  const normalized = stripIpv6Brackets(hostname).toLowerCase()
  return normalized === "127.0.0.1" || normalized === "localhost" || normalized === "::1"
}

function stripIpv6Brackets(hostname: string): string {
  return hostname.replace(/^\[|\]$/g, "")
}

function normalizeFilesystemPath(value: string): string {
  return value.replaceAll("\\", "/")
}

function formatUnknownError(error: unknown): string {
  if (error instanceof Error) {
    return error.message
  }
  return String(error ?? "unknown error")
}
