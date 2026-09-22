export interface WishlistMetadata {
  title: string; summary: string; actors: string[]; tags: string[]; studio: string
  releaseDate: string; runtimeMinutes: number; provider: string; homepage: string
}
export interface WishlistAsset { id: string; role: string; url: string; thumbnailUrl: string }
export interface WishlistItem {
  id: string; code: string; metadata: WishlistMetadata; note: string; completed: boolean
  status: "pending" | "completed" | "in_library"
  enrichmentState: "queued" | "running" | "ready" | "partial" | "failed" | "needs_review"
  error: string; version: number; createdAt: string; updatedAt: string; movieIds: string[]; assets: WishlistAsset[]
}
export interface WishlistQuery { status?: string; q?: string; cursor?: string; limit?: number }
export interface WishlistPage { items: WishlistItem[]; total: number; pendingCount: number; nextCursor?: string }
export interface WishlistPatch { version: number; code?: string; note?: string; completed?: boolean }
