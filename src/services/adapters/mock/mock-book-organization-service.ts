import type { BookKind, BookOrganizationJob, BookOrganizationItem, BookOrganizationService, BookOrganizationRequest } from "@/services/contracts/book-organization-service"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import { mockComicLibraryService, applyMockComicMetadata } from "./mock-comic-library-service"
import { mockPhotoLibraryService, applyMockPhotoMetadata } from "./mock-photo-library-service"

const key = "curated-mock-book-organization-v1"
type Record = { job: BookOrganizationJob; request: BookOrganizationRequest; items: BookOrganizationItem[] }
function read(): Record[] { try { return JSON.parse(localStorage.getItem(key) ?? "[]") } catch { return [] } }
function save(records: Record[]) { localStorage.setItem(key, JSON.stringify(records)) }
function permitted() { if (!useExperimentalAgent().writeEnabled.value) throw new Error("AI_READ_ONLY") }
function bookState(kind: BookKind, id: string): BookOrganizationItem["before"] {
  const book = kind === "comic" ? mockComicLibraryService.getComicById(id) : mockPhotoLibraryService.getPhotoById(id)
  if (!book) throw new Error("BOOK_NOT_FOUND")
  return { title: book.title, originalTitle: book.originalTitle || book.title, sourceFileName: book.sourceFileName, metadata: book.metadata }
}
function recount(record: Record) {
  const { job, items } = record
  job.processed = items.filter(i => i.status !== "pending").length
  job.pendingReview = items.filter(i => i.status === "ready").length
  job.applied = items.filter(i => i.status === "applied").length
  job.failed = items.filter(i => ["failed", "conflict"].includes(i.status)).length
  job.status = job.pendingReview ? "needs_confirmation" : job.failed ? "partial_failed" : "completed"
  job.stage = job.pendingReview ? "reviewing" : "done"
  job.updatedAt = new Date().toISOString()
}
function simulate(record: Record) {
  for (const item of record.items.filter(i => ["pending", "failed", "conflict"].includes(i.status))) {
    item.revision++; item.reason = ""
    try {
      item.before = bookState(record.job.kind, item.bookId)
      item.metadata = { cleanTitle: item.before.originalTitle, translatedTitle: `Mock: ${item.before.originalTitle}`, uncertainty: "Mock — no real translation or identity recognition." }
      item.status = "ready"
    } catch { item.status = "failed"; item.reason = "BOOK_RECOGNITION_FAILED" }
  }
  recount(record)
}
export function listMockBookOrganizations() { return read().map(record => record.job) }
export const mockBookOrganizationService: BookOrganizationService = {
  async start(request) {
    permitted()
    const records = read(), existing = records.find(r => r.request.requestId === request.requestId)
    if (existing) {
      if (JSON.stringify(existing.request) !== JSON.stringify(request)) throw new Error("AI_WRITE_CONFLICT")
      return existing.job
    }
    if (!request.bookIds.length || request.bookIds.length > 600 || new Set(request.bookIds).size !== request.bookIds.length) throw new Error("BAD_REQUEST")
    const now = new Date().toISOString()
    const job: BookOrganizationJob = { id: crypto.randomUUID(), kind: request.kind, locale: request.locale, status: "queued", stage: "queued", total: request.bookIds.length, processed: 0, pendingReview: 0, applied: 0, failed: 0, createdAt: now, updatedAt: now }
    const record: Record = { job, request, items: request.bookIds.map(bookId => { const before = bookState(request.kind, bookId); return { bookId, title: before.title, before, metadata: { cleanTitle: "", translatedTitle: "" }, status: "pending", revision: 0, reason: "" } }) }
    // Mock resolves immediately and persists reviewable results; reads never generate work.
    simulate(record); records.push(record); save(records)
    return job
  },
  async items(id, offset = 0) { const record = read().find(r => r.job.id === id); if (!record) throw new Error("TASK_NOT_FOUND"); return record.items.slice(offset, offset + 25) },
  async control(id, operation) {
    permitted()
    const records = read(), record = records.find(r => r.job.id === id)
    if (!record) throw new Error("TASK_NOT_FOUND")
    if (operation === "cancel") { if (!["queued", "running"].includes(record.job.status)) throw new Error("AI_WRITE_CONFLICT"); record.job.status = "cancelled" }
    else { if (["queued", "running", "completed"].includes(record.job.status)) throw new Error("AI_WRITE_CONFLICT"); simulate(record) }
    save(records)
  },
  async review(id, review) {
    permitted()
    const records = read(), record = records.find(r => r.job.id === id), item = record?.items.find(i => i.bookId === review.bookId)
    if (!record || !item) throw new Error("TASK_NOT_FOUND")
    if (item.revision !== review.revision) throw new Error("AI_WRITE_CONFLICT")
    if (item.status === (review.decision === "apply" ? "applied" : "discarded")) return
    if (item.status !== "ready") throw new Error("AI_WRITE_CONFLICT")
    if (review.decision === "discard") item.status = "discarded"
    else {
      if (JSON.stringify(bookState(record.job.kind, item.bookId)) !== JSON.stringify(item.before)) {
        item.status = "conflict"; item.reason = "AI_WRITE_CONFLICT"; recount(record); save(records); throw new Error("AI_WRITE_CONFLICT")
      }
      if (record.job.kind === "comic") await applyMockComicMetadata(item.bookId, item.metadata)
      else await applyMockPhotoMetadata(item.bookId, item.metadata)
      item.status = "applied"
    }
    recount(record); save(records)
  },
}
