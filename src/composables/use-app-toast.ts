import { toast } from "vue-sonner"
import { h } from "vue"
import { Loader2Icon } from "lucide-vue-next"
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

const DEFAULT_TOAST_DURATION_MS = 4500

function toastDuration(durationMs?: number): number {
  return durationMs !== undefined && Number.isFinite(durationMs) && durationMs > 0 && durationMs <= 2_147_483_647
    ? durationMs
    : DEFAULT_TOAST_DURATION_MS
}

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
  action?: { label: string; onClick: () => void }
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
  // Sonner's loading type never auto-closes, even with a finite duration.
  return toast(message, {
    icon: h(Loader2Icon, { class: "size-4 animate-spin", "aria-hidden": "true" }),
    duration: DEFAULT_TOAST_DURATION_MS,
    closeButton: true,
    dismissible: true,
  })
}

export function dismissAppToast(id: AppToastId): void {
  toast.dismiss(id)
}

export function pushAppToast(message: string, options?: PushAppToastOptions): void {
  const variant = options?.variant ?? "default"
  const duration = toastDuration(options?.durationMs)
  const base = {
    duration,
    // Sonner merges updates by ID; clear a previous progress indicator.
    icon: undefined,
    closeButton: true,
    dismissible: true,
    ...(options?.id === undefined ? {} : { id: options.id }),
    ...(options?.action === undefined ? {} : { action: options.action }),
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
