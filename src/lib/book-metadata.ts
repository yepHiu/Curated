import type { BookMetadata } from "@/api/types"

export const bookCreditRoles = ["authors", "circles", "models", "photographers"] as const

export function bookMetadataSearchText(metadata?: BookMetadata): string {
  if (!metadata) return ""
  return [metadata.cleanTitle, metadata.translatedTitle, ...bookCreditRoles.flatMap(role => metadata[role]?.map(item => item.name) ?? [])].filter(Boolean).join(" ")
}

export function isBookMetadata(value: unknown): value is BookMetadata {
  if (!value || typeof value !== "object" || Array.isArray(value)) return false
  const m = value as Record<string, unknown>
  if (typeof m.cleanTitle !== "string" || typeof m.translatedTitle !== "string") return false
  if (m.uncertainty !== undefined && typeof m.uncertainty !== "string") return false
  return bookCreditRoles.every(role => m[role] === undefined || (Array.isArray(m[role]) && m[role].every(item => item && typeof item === "object" && typeof item.name === "string" && typeof item.evidence === "string")))
}
