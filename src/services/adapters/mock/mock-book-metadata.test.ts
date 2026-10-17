import { beforeEach, expect, it, vi } from "vitest"

beforeEach(() => { localStorage.clear(); vi.resetModules() })

it.each(["comic", "photo"] as const)("previews and persists confirmed %s metadata without changing source or tags", async kind => {
  const { mockAIService: ai } = await import("./mock-ai-service")
  const { mockComicLibraryService: comics } = await import("./mock-comic-library-service")
  const { mockPhotoLibraryService: photos } = await import("./mock-photo-library-service")
  const book = kind === "comic" ? comics.comics.value[0]! : photos.photos.value[0]!
  const original = { title: book.title, file: book.sourceFileName, tags: [...book.tags] }
  const preview = await ai.runAction("recognize_book_metadata", { [kind === "comic" ? "comicId" : "photoId"]: book.id })
  expect(book.title).toBe(original.title)
  const request = { name: preview.name, sessionId: preview.sessionId, arguments: preview.arguments!, confirmToken: preview.confirmToken! }
  await ai.confirmTool(request)
  const saved = kind === "comic" ? comics.getComicById(book.id)! : photos.getPhotoById(book.id)!
  expect(saved.title).toBe(preview.proposedText)
  expect(saved.metadata?.cleanTitle).toBe(original.title)
  expect(saved.sourceFileName).toBe(original.file)
  expect(saved.tags).toEqual(original.tags)
  const { readMockBookMetadata } = await import("@/lib/mock-book-metadata")
  expect(readMockBookMetadata(kind, book.id)).toEqual(saved.metadata)
  await expect(ai.confirmTool(request)).rejects.toMatchObject({ code: "AI_WRITE_CONFLICT" })
})
