import { toast } from "vue-sonner"
import type { TaskDTO } from "@/api/types"

export type AppToastVariant = "default" | "success" | "destructive" | "warning"

/** Maps terminal task status to toast severity (scan / watch / scrape toasts). */
export function taskTerminalToastVariant(status: TaskDTO["status"]): AppToastVariant {
  if (status === "completed") {
    return "success"
  }
  if (status === "partial_failed") {
    return "warning"
  }
  if (status === "cancelled") {
    return "default"
  }
  return "destructive"
}

export function pushAppToast(
  message: string,
  options?: { variant?: AppToastVariant; durationMs?: number },
): void {
  const variant = options?.variant ?? "default"
  const duration = options?.durationMs ?? 4500
  const base = {
    duration,
    closeButton: true,
    dismissible: true,
  } as const

  switch (variant) {
    case "success":
      toast.success(message, base)
      break
    case "destructive":
      toast.error(message, { ...base, important: true })
      break
    case "warning":
      toast.warning(message, base)
      break
    default:
      toast.info(message, base)
      break
  }
}
