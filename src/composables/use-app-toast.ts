import { toast } from "vue-sonner"
import type { TaskDTO } from "@/api/types"
import {
  useNotificationCenter,
  type NotificationType,
  type NotificationSource,
} from "@/composables/use-notification-center"

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

export interface PushAppToastOptions {
  variant?: AppToastVariant
  durationMs?: number
  /** 如果传入，则同时写入通知中心持久化记录 */
  notification?: {
    type: NotificationType
    title: string
    source?: NotificationSource
  }
}

export function pushAppToast(message: string, options?: PushAppToastOptions): void {
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

  if (options?.notification) {
    const severity =
      variant === "destructive" ? "error" : variant === "warning" ? "warning" : variant === "success" ? "success" : "info"
    useNotificationCenter().addNotification({
      type: options.notification.type,
      severity,
      title: options.notification.title,
      message,
      source: options.notification.source,
    })
  }
}
