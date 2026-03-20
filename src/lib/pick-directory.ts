/**
 * 选择「视频库」目录。
 * - 桌面壳（Electron 等）可通过 `window.javLibrary.pickDirectory()` 返回绝对路径。
 * - Chromium：优先 `showDirectoryPicker()`（仅能拿到文件夹名，无法得到磁盘绝对路径）。
 * - 回退：`<input webkitdirectory>`，在部分壳环境里 `File.path` 存在时可解析出目录。
 */

export type PickDirectoryOutcome =
  | { status: "ok"; path: string }
  | { status: "hint"; message: string; suggestedTitle?: string }
  | { status: "cancelled" }
  | { status: "unsupported" }

type WindowWithPickers = Window & {
  javLibrary?: {
    pickDirectory?: () => Promise<{ path: string } | null | undefined>
  }
  showDirectoryPicker?: (options?: {
    mode?: "read" | "readwrite"
  }) => Promise<FileSystemDirectoryHandle>
}

function directoryFromFilePath(filePath: string): string {
  const back = filePath.lastIndexOf("\\")
  const fwd = filePath.lastIndexOf("/")
  const i = Math.max(back, fwd)
  return i > 0 ? filePath.slice(0, i) : filePath
}

function pickDirectoryViaFileInput(): Promise<PickDirectoryOutcome> {
  return new Promise((resolve) => {
    const input = document.createElement("input")
    input.type = "file"
    input.webkitdirectory = true
    input.setAttribute("directory", "")
    input.multiple = true
    input.style.cssText = "position:fixed;width:0;height:0;opacity:0;pointer-events:none"

    const finish = (out: PickDirectoryOutcome) => {
      input.remove()
      resolve(out)
    }

    input.addEventListener("change", () => {
      const files = input.files
      if (!files?.length) {
        finish({ status: "cancelled" })
        return
      }
      const f = files[0] as File & { path?: string }
      if (f.path && typeof f.path === "string") {
        finish({ status: "ok", path: directoryFromFilePath(f.path) })
        return
      }
      const rel = files[0].webkitRelativePath
      const seg = rel.includes("/") ? rel.slice(0, rel.indexOf("/")) : rel
      finish({
        status: "hint",
        suggestedTitle: seg || undefined,
        message:
          "当前环境无法自动读取磁盘绝对路径。请在本机资源管理器中打开该文件夹，将地址栏中的完整路径复制到「绝对路径」输入框。",
      })
    })

    input.addEventListener("cancel", () => {
      finish({ status: "cancelled" })
    })

    document.body.appendChild(input)
    input.click()
  })
}

export async function pickLibraryDirectory(): Promise<PickDirectoryOutcome> {
  const w = window as WindowWithPickers

  if (w.javLibrary?.pickDirectory) {
    try {
      const r = await w.javLibrary.pickDirectory()
      if (r?.path?.trim()) {
        return { status: "ok", path: r.path.trim() }
      }
    } catch {
      /* ignore */
    }
    return { status: "cancelled" }
  }

  if (typeof w.showDirectoryPicker === "function") {
    try {
      const handle = await w.showDirectoryPicker({ mode: "read" })
      const name = handle.name
      return {
        status: "hint",
        suggestedTitle: name,
        message: `已选择文件夹「${name}」。网页出于安全限制无法读取本机绝对路径，请在资源管理器中进入该文件夹，将地址栏路径复制到上方「绝对路径」。`,
      }
    } catch (e) {
      const name = (e as { name?: string }).name
      if (name === "AbortError") {
        return { status: "cancelled" }
      }
      // 权限 / 非安全上下文等：尝试传统选择器
    }
  }

  try {
    return await pickDirectoryViaFileInput()
  } catch {
    return { status: "unsupported" }
  }
}
