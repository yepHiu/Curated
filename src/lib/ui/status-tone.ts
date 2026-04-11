export type StatusTone = "success" | "warning" | "danger" | "info"

const statusTextClasses: Record<StatusTone, string> = {
  success: "text-success",
  warning: "text-warning",
  danger: "text-danger",
  info: "text-info",
}

const statusDotClasses: Record<StatusTone, string> = {
  success: "bg-success shadow-sm ring-1 ring-success/40",
  warning: "bg-warning shadow-sm ring-1 ring-warning/40",
  danger: "bg-danger shadow-sm ring-1 ring-danger/40",
  info: "bg-info shadow-sm ring-1 ring-info/40",
}

const statusBadgeClasses: Record<StatusTone, string> = {
  success: "border-success/35 bg-success/10 text-success",
  warning: "border-warning/35 bg-warning/10 text-warning",
  danger: "border-danger/35 bg-danger/10 text-danger",
  info: "border-info/35 bg-info/10 text-info",
}

const statusPanelClasses: Record<StatusTone, string> = {
  success:
    "rounded-2xl border border-success/25 border-l-[3px] border-l-success/60 bg-success/[0.07] px-4 py-3",
  warning:
    "rounded-2xl border border-warning/25 border-l-[3px] border-l-warning/60 bg-warning/[0.07] px-4 py-3",
  danger:
    "rounded-2xl border border-danger/25 border-l-[3px] border-l-danger/60 bg-danger/[0.07] px-4 py-3",
  info:
    "rounded-2xl border border-info/25 border-l-[3px] border-l-info/60 bg-info/[0.07] px-4 py-3",
}

export function statusTextClass(tone: StatusTone): string {
  return statusTextClasses[tone]
}

export function statusDotClass(tone: StatusTone): string {
  return statusDotClasses[tone]
}

export function statusBadgeClass(tone: StatusTone): string {
  return statusBadgeClasses[tone]
}

export function statusPanelClass(tone: StatusTone): string {
  return statusPanelClasses[tone]
}
