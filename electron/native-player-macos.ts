import { createRequire } from "node:module"
import { EventEmitter } from "node:events"
import type { NativeEngineLauncher } from "./native-mpv-player.js"

interface MacNativeBinding {
  create(view: Buffer, library: string, args: string[]): unknown
  poll(instance: unknown): boolean
  stop(instance: unknown): void
}

/** NSView/引擎路径仅由主进程提供；renderer 沿用受限的 playback bridge。 */
export function macNativeLauncher(modulePath: string, view: Buffer): NativeEngineLauncher {
  const binding = createRequire(import.meta.url)(modulePath) as MacNativeBinding
  return (library, args) => {
    const instance = binding.create(view, library, args)
    const engine = new EventEmitter() as EventEmitter & { kill(): void }
    let stopped = false
    const stop = () => {
      if (stopped) return
      stopped = true
      clearInterval(timer)
      binding.stop(instance)
      engine.emit("exit", 0)
    }
    const timer = setInterval(() => { if (binding.poll(instance)) stop() }, 20)
    engine.kill = stop
    return engine
  }
}
