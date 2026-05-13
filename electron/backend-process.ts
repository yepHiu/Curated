import { spawn, type ChildProcessByStdio } from "node:child_process"
import { existsSync } from "node:fs"
import path from "node:path"
import type { Readable } from "node:stream"
import { setTimeout as delay } from "node:timers/promises"

type ElectronEnv = Partial<Record<string, string | undefined>>

export interface BackendLaunchPlan {
  command: string
  args: string[]
  cwd: string
}

type BackendChildProcess = ChildProcessByStdio<null, Readable, Readable>

interface BackendLaunchPlanOptions {
  appPath: string
  env?: ElectronEnv
  isWindows?: boolean
  pathExists?: (candidate: string) => boolean
}

interface StartBackendOptions extends BackendLaunchPlanOptions {
  baseUrl?: string
  isPackaged: boolean
  fetchImpl?: typeof fetch
  timeoutMs?: number
}

export interface ManagedBackend {
  baseUrl: string
  attachedToExisting: boolean
  process?: BackendChildProcess
  stop: () => Promise<void>
}

const backendUrlEnvKeys = ["CURATED_ELECTRON_BACKEND_URL", "CURATED_BACKEND_URL"] as const
const backendPathEnvKey = "CURATED_ELECTRON_BACKEND_PATH"

export function defaultBackendBaseUrl(isPackaged: boolean): string {
  return `http://127.0.0.1:${isPackaged ? "8081" : "8080"}`
}

export function resolveBackendBaseUrl(env: ElectronEnv = process.env, isPackaged: boolean): string {
  for (const key of backendUrlEnvKeys) {
    const value = env[key]?.trim()
    if (value) {
      return normalizeBackendBaseUrl(value)
    }
  }
  return defaultBackendBaseUrl(isPackaged)
}

export function resolveBackendLaunchPlan(options: BackendLaunchPlanOptions): BackendLaunchPlan {
  const env = options.env ?? process.env
  const isWindows = options.isWindows ?? process.platform === "win32"
  const pathExists = options.pathExists ?? existsSync
  const executableName = isWindows ? "curated.exe" : "curated"
  const explicitBackendPath = env[backendPathEnvKey]?.trim()
  const candidates = [
    explicitBackendPath,
    joinAppPath(options.appPath, "backend", "runtime", isWindows ? "curated-dev.exe" : "curated-dev"),
    joinAppPath(options.appPath, "release", "Curated", executableName),
    joinAppPath(options.appPath, "backend", executableName),
  ].filter((candidate): candidate is string => Boolean(candidate))

  const binaryPath = candidates.find((candidate) => pathExists(candidate))
  if (binaryPath) {
    return {
      command: binaryPath,
      args: ["-mode", "http"],
      cwd: resolveBinaryWorkingDirectory(options.appPath, binaryPath),
    }
  }

  return {
    command: "go",
    args: ["run", "./cmd/curated", "-mode", "http"],
    cwd: joinAppPath(options.appPath, "backend"),
  }
}

export async function startBackend(options: StartBackendOptions): Promise<ManagedBackend> {
  const fetchImpl = options.fetchImpl ?? fetch
  const baseUrl = options.baseUrl ?? resolveBackendBaseUrl(options.env, options.isPackaged)

  if (await isBackendHealthy(baseUrl, fetchImpl)) {
    return {
      baseUrl,
      attachedToExisting: true,
      stop: async () => {},
    }
  }

  const plan = resolveBackendLaunchPlan(options)
  const backendProcess = spawn(plan.command, plan.args, {
    cwd: plan.cwd,
    env: {
      ...process.env,
      ...options.env,
      CURATED_HOSTED_BY: "electron",
    },
    stdio: ["ignore", "pipe", "pipe"],
    windowsHide: true,
  })

  backendProcess.stdout.on("data", (chunk: Buffer) => {
    process.stdout.write(`[curated-backend] ${chunk.toString()}`)
  })
  backendProcess.stderr.on("data", (chunk: Buffer) => {
    process.stderr.write(`[curated-backend] ${chunk.toString()}`)
  })

  try {
    await waitForBackendHealth(baseUrl, fetchImpl, options.timeoutMs ?? 15_000)
  } catch (error) {
    await stopBackendProcess(backendProcess)
    throw error
  }

  return {
    baseUrl,
    attachedToExisting: false,
    process: backendProcess,
    stop: () => stopBackendProcess(backendProcess),
  }
}

async function waitForBackendHealth(baseUrl: string, fetchImpl: typeof fetch, timeoutMs: number): Promise<void> {
  const deadline = Date.now() + timeoutMs
  let lastError: unknown
  while (Date.now() < deadline) {
    try {
      if (await isBackendHealthy(baseUrl, fetchImpl)) {
        return
      }
    } catch (error) {
      lastError = error
    }
    await delay(250)
  }
  throw new Error(`Curated backend did not become ready at ${baseUrl}: ${formatUnknownError(lastError)}`)
}

async function isBackendHealthy(baseUrl: string, fetchImpl: typeof fetch): Promise<boolean> {
  try {
    const response = await fetchImpl(`${baseUrl}/api/health`, { signal: AbortSignal.timeout(1000) })
    if (!response.ok) {
      return false
    }
    return isCuratedHealthPayload(await response.json())
  } catch {
    return false
  }
}

export function isCuratedHealthPayload(payload: unknown): boolean {
  if (!payload || typeof payload !== "object") {
    return false
  }
  const name = "name" in payload ? String(payload.name).trim().toLowerCase() : ""
  return name === "curated" || name === "curated-dev"
}

async function stopBackendProcess(backendProcess: BackendChildProcess): Promise<void> {
  if (backendProcess.exitCode !== null || backendProcess.killed) {
    return
  }

  await new Promise<void>((resolve) => {
    const timeout = setTimeout(() => {
      if (backendProcess.exitCode === null && !backendProcess.killed) {
        backendProcess.kill("SIGKILL")
      }
      resolve()
    }, 3000)

    backendProcess.once("exit", () => {
      clearTimeout(timeout)
      resolve()
    })
    backendProcess.kill()
  })
}

function normalizeBackendBaseUrl(rawUrl: string): string {
  const parsed = new URL(rawUrl)
  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new Error(`Unsupported Curated backend URL protocol: ${parsed.protocol}`)
  }
  if (parsed.hostname === "localhost") {
    parsed.hostname = "127.0.0.1"
  }
  return parsed.origin
}

function joinAppPath(appPath: string, ...segments: string[]): string {
  return normalizeFilesystemPath(path.join(appPath, ...segments))
}

function resolveBinaryWorkingDirectory(appPath: string, binaryPath: string): string {
  const normalizedBinaryPath = normalizeFilesystemPath(binaryPath)
  const devRuntimeDir = joinAppPath(appPath, "backend", "runtime")
  if (normalizedBinaryPath.startsWith(`${devRuntimeDir}/`)) {
    return joinAppPath(appPath, "backend")
  }
  return normalizeFilesystemPath(path.dirname(binaryPath))
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
