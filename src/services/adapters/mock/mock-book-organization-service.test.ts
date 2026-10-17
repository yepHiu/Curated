import { beforeEach, expect, it, vi } from "vitest"
import { projectBookOrganization, summarizeTasks } from "@/lib/task-center"

beforeEach(() => { vi.stubEnv("VITE_USE_WEB_API", "false"); localStorage.clear(); vi.resetModules(); localStorage.setItem("curated-agent-experimental-v1", "true") })
it.each(["comic", "photo"] as const)("persists %s proposals through reload, projects attention and protects manual edits", async kind => {
  let { mockBookOrganizationService: service } = await import("./mock-book-organization-service")
  const { mockComicLibraryService: comics } = await import("./mock-comic-library-service")
  const { mockPhotoLibraryService: photos } = await import("./mock-photo-library-service")
  const book = kind === "comic" ? comics.comics.value[0]! : photos.photos.value[0]!
  const req = { kind, bookIds: [book.id], locale: "zh-CN", requestId: "test" }
  const job = await service.start(req)
  expect(book.title).not.toContain("Mock:")
  expect(summarizeTasks([projectBookOrganization(job)]).attention).toBe(1)
  expect((await service.start(req)).id).toBe(job.id)
  vi.resetModules()
  ;({ mockBookOrganizationService: service } = await import("./mock-book-organization-service"))
  const item = (await service.items(job.id))[0]!
  await expect(service.review(job.id, { bookId: book.id, revision: 0, decision: "apply" })).rejects.toThrow("AI_WRITE_CONFLICT")
  await service.review(job.id, { bookId: book.id, revision: item.revision, decision: "apply" })
  expect((await service.items(job.id))[0]!.status).toBe("applied")
  const c = (await import("./mock-comic-library-service")).mockComicLibraryService
  const p = (await import("./mock-photo-library-service")).mockPhotoLibraryService
  if (kind === "comic") await c.patchComic(book.id, { title: "Manual" })
  else await p.patchPhoto(book.id, { title: "Manual" })
  await service.review(job.id, { bookId: book.id, revision: item.revision, decision: "apply" })
  expect(kind === "comic" ? c.getComicById(book.id)!.title : p.getPhotoById(book.id)!.title).toBe("Manual")
  const next = await service.start({ ...req, requestId: "next" })
  if (kind === "comic") await c.patchComic(book.id, { title: "Later" })
  else await p.patchPhoto(book.id, { title: "Later" })
  await expect(service.review(next.id, { bookId: book.id, revision: 1, decision: "apply" })).rejects.toThrow("AI_WRITE_CONFLICT")
  expect((await service.items(next.id))[0]!.status).toBe("conflict")
  await service.control(next.id, "retry")
  expect((await service.items(next.id))[0]!.revision).toBe(2)
})
