import type { BookMetadata } from "@/api/types"

export type BookKind = "comic" | "photo"
export interface BookOrganizationRequest { kind: BookKind; bookIds: string[]; locale: string; requestId: string }
export interface BookOrganizationJob {
  id: string; kind: BookKind; locale: string; status: string; stage: string
  total: number; processed: number; pendingReview: number; applied: number; failed: number
  createdAt: string; updatedAt: string
}
export interface BookOrganizationItem {
  bookId: string; title: string; status: "pending" | "ready" | "applied" | "discarded" | "unchanged" | "failed" | "conflict"; revision: number
  before: { title: string; originalTitle: string; sourceFileName: string; metadata?: BookMetadata }
  metadata: BookMetadata; reason: string
}
export interface BookOrganizationReview { bookId: string; revision: number; decision: "apply" | "discard" }
export interface BookOrganizationService {
  start(request: BookOrganizationRequest): Promise<BookOrganizationJob>
  items(id: string, offset?: number): Promise<BookOrganizationItem[]>
  control(id: string, operation: "cancel" | "retry"): Promise<unknown>
  review(id: string, review: BookOrganizationReview): Promise<unknown>
}
