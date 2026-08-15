import { toast } from "vue-sonner"
import type { TaskDTO } from "@/api/types"
import {
  useNotificationCenter,
  type NotificationType,
  type NotificationSource,
} from "@/composables/use-notification-center"
import {
  shouldPersistToMessageCenter,
  type MessagePolicyId,
} from "@/lib/message-policy"

export type AppToastVariant = "default" | "success" | "destructive" | "warning"
export type AppToastId = string | number

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
  /** Replaces an existing toast, such as a loading indicator for the same action. */
  id?: AppToastId
  /** 如果传入，则按消息政策台账决定是否写入消息中心 */
  notification?: {
    messageId: MessagePolicyId
    type: NotificationType
    title: string
    source?: NotificationSource
    group?: string
  }
}

export function pushAppToastLoading(message: string): AppToastId {
  return toast.loading(message, {
    duration: Number.POSITIVE_INFINITY,
    closeButton: true,
    dismissible: true,
  })
}

export function dismissAppToast(id: AppToastId): void {
  toast.dismiss(id)
}

export function pushAppToast(message: string, options?: PushAppToastOptions): void {
  const variant = options?.variant ?? "default"
  const duration = options?.durationMs ?? 4500
  const base = {
    duration,
    closeButton: true,
    dismissible: true,
    ...(options?.id === undefined ? {} : { id: options.id }),
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

  if (options?.notification && shouldPersistToMessageCenter(options.notification.messageId)) {
    const severity =
      variant === "destructive"
        ? "error"
        : variant === "warning"
          ? "warning"
          : variant === "success"
            ? "success"
            : "info"
    useNotificationCenter().addNotification({
      messageId: options.notification.messageId,
      type: options.notification.type,
      severity,
      title: options.notification.title,
      message,
      source: options.notification.source,
      group: options.notification.group,
    })
  }
}
