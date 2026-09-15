import type { AIConfirmChangeDTO, SavedViewFiltersV1 } from "@/api/types"
import { summarizeSavedViewFilters } from "@/lib/saved-view-summary"

export type AgentConfirmTranslate = (key: string, values?: Record<string, unknown>) => string

export type AgentConfirmCopy = {
  title: string
  paragraphs: string[]
  applyLabel: string
  appliedLabel: string
  discardedLabel: string
  showRawChanges: boolean
}

export type AgentConfirmSource = {
  name: string
  arguments: Record<string, unknown>
  changes: AIConfirmChangeDTO[]
}

function changeAt(changes: AIConfirmChangeDTO[], path: string): AIConfirmChangeDTO | undefined {
  return changes.find((item) => item.path === path)
}

function asTrimmedString(value: unknown): string {
  return typeof value === "string" ? value.trim() : ""
}

function asSavedViewFilters(value: unknown): SavedViewFiltersV1 | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null
  const rec = value as Record<string, unknown>
  if (rec.schemaVersion !== undefined && rec.schemaVersion !== 1) return null
  return { ...rec, schemaVersion: 1 } as SavedViewFiltersV1
}

function savedViewName(source: AgentConfirmSource): string {
  const fromChange = asTrimmedString(changeAt(source.changes, "savedView.name")?.after)
  if (fromChange) return fromChange
  return asTrimmedString(source.arguments.name)
}

export function savedViewFiltersFromConfirm(source: AgentConfirmSource): SavedViewFiltersV1 | null {
  return (
    asSavedViewFilters(changeAt(source.changes, "savedView.filters")?.after) ??
    asSavedViewFilters(source.arguments.filters)
  )
}

function asText(value: unknown): string {
  if (value == null) return ""
  if (typeof value === "string") return value
  if (typeof value === "number" || typeof value === "boolean") return String(value)
  return ""
}

function commentCopy(source: AgentConfirmSource, t: AgentConfirmTranslate): string[] {
  const change = changeAt(source.changes, "comment.body")
  const before = asText(change?.before)
  const after = asText(change?.after)
  if (!before && after) {
    return [t("agentWindow.confirmCommentCreate", { body: after })]
  }
  if (before && !after) {
    return [t("agentWindow.confirmCommentClear")]
  }
  if (after) {
    return [t("agentWindow.confirmCommentReplace", { body: after })]
  }
  return []
}

function displayCopy(source: AgentConfirmSource, t: AgentConfirmTranslate): string[] {
  const paragraphs: string[] = []
  const title = changeAt(source.changes, "display.userTitle")
  const summary = changeAt(source.changes, "display.userSummary")
  const nextTitle = asText(title?.after)
  const nextSummary = asText(summary?.after)
  if (title) {
    paragraphs.push(
      nextTitle
        ? t("agentWindow.confirmDisplayTitle", { after: nextTitle })
        : t("agentWindow.confirmDisplayTitleClear"),
    )
  }
  if (summary) {
    paragraphs.push(
      nextSummary
        ? t("agentWindow.confirmDisplaySummary", { after: nextSummary })
        : t("agentWindow.confirmDisplaySummaryClear"),
    )
  }
  return paragraphs
}

export function agentConfirmCopy(source: AgentConfirmSource, t: AgentConfirmTranslate): AgentConfirmCopy {
  const discardedLabel = t("agentWindow.confirmDiscarded")
  if (source.name === "create_saved_view") {
    const name = savedViewName(source)
    const filters = savedViewFiltersFromConfirm(source)
    const summary = filters ? summarizeSavedViewFilters(filters, t) : ""
    const paragraphs = [
      t("agentWindow.confirmCreateViewLead", { name: name || t("agentWindow.sessionPlaceholder") }),
    ]
    if (summary) {
      paragraphs.push(t("agentWindow.confirmCreateViewFilters", { summary }))
    }
    return {
      title: t("agentWindow.confirmTitleCreateView"),
      paragraphs,
      applyLabel: t("agentWindow.confirmApplyCreateView"),
      appliedLabel: t("agentWindow.confirmAppliedCreateView"),
      discardedLabel,
      showRawChanges: false,
    }
  }
  if (source.name === "save_movie_comment" || source.name === "save_comic_comment" || source.name === "save_photo_comment") {
    const paragraphs = commentCopy(source, t)
    return {
      title: t("agentWindow.confirmTitleComment"),
      paragraphs,
      applyLabel: t("agentWindow.confirmApply"),
      appliedLabel: t("agentWindow.confirmApplied"),
      discardedLabel,
      showRawChanges: paragraphs.length === 0,
    }
  }
  if (source.name === "update_movie_display_overrides" || source.name === "update_comic_title" || source.name === "update_photo_title") {
    const paragraphs = displayCopy(source, t)
    return {
      title: t("agentWindow.confirmTitleDisplay"),
      paragraphs,
      applyLabel: t("agentWindow.confirmApply"),
      appliedLabel: t("agentWindow.confirmApplied"),
      discardedLabel,
      showRawChanges: paragraphs.length === 0,
    }
  }
  return {
    title: t("agentWindow.confirmTitle"),
    paragraphs: [],
    applyLabel: t("agentWindow.confirmApply"),
    appliedLabel: t("agentWindow.confirmApplied"),
    discardedLabel,
    showRawChanges: true,
  }
}
