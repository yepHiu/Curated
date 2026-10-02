// @vitest-environment jsdom
import { beforeEach, expect, it, vi } from "vitest"
import type { Movie } from "@/domain/movie/types"

beforeEach(() => { localStorage.clear(); vi.resetModules() })

it("counts no-match analysis, persists coverage, and selects changed sources separately", async () => {
 let { createMockTopicServices } = await import("./mock-topic-service")
 const movies = [{ id: "a", title: "A", summary: "", tags: ["Theme"], userTags: [] }, { id: "b", title: "B", summary: "", tags: [], userTags: [] }, { id: "trash", title: "Trash", tags: [], userTags: [], trashedAt: "2026-10-03" }] as unknown as Movie[]
 const patch = vi.fn((id, tags) => { movies.find((movie) => movie.id === id)!.userTags = tags })
 let service = createMockTopicServices(() => movies, patch)
 expect(await service.getTagOrganizationStats()).toEqual({ total: 2, organized: 0, unorganized: 2, outdated: 0, unresolved: 0 })
 const job = await service.startTagOrganization("unorganized")
 expect(job.total).toBe(2)
 expect(await service.getTagOrganizationStats()).toEqual({ total: 2, organized: 2, unorganized: 0, outdated: 0, unresolved: 1 })
 vi.resetModules(); ({ createMockTopicServices } = await import("./mock-topic-service"))
 service = createMockTopicServices(() => movies, patch)
 await expect(service.startTagOrganization("unorganized")).rejects.toThrow("AI_ORGANIZATION_NO_MOVIES")
 movies[0]!.summary = "Changed"
 expect((await service.getTagOrganizationStats()).outdated).toBe(1)
 expect((await service.startTagOrganization("outdated")).total).toBe(1)
 expect((await service.getTagOrganizationStats()).organized).toBe(2)
 // Undoing the older write must preserve the more recent no-change analysis.
 await service.undoTagOrganization(job.id)
 expect((await service.getTagOrganizationStats()).organized).toBe(2)
})
it("writes only user tags, exposes evidence, and undo preserves later manual edits", async () => {
 const { createMockTopicServices } = await import("./mock-topic-service")
 const movies = [{ id: "a", title: "A", tags: ["Theme"], userTags: [] }, { id: "b", title: "B", tags: ["Theme"], userTags: [] }] as unknown as Movie[]
 const service = createMockTopicServices(() => movies, (id, tags) => { movies.find((m) => m.id === id)!.userTags = tags })
 expect(await service.getHomepageTopics()).toEqual([])
 const job = await service.startTagOrganization("all")
 expect((await service.getHomepageTopics())[0]?.movies).toHaveLength(2)
 expect((await service.getTagOrganizationItems(job.id))[0]?.evidence[0]?.quote).toBe("Theme")
 movies[0]!.userTags = ["Manual"]
 expect(await service.undoTagOrganization(job.id)).toEqual({ restored: 1, conflicts: 1 })
 expect(movies.map((m) => m.tags)).toEqual([["Theme"], ["Theme"]])
 expect(movies[0]!.userTags).toEqual(["Manual"])
 expect((await service.getHomepageTopics())).toEqual([])
})

it("organizes only the selected movies and records the selected count", async () => {
 const { createMockTopicServices } = await import("./mock-topic-service")
 const movies = [{ id: "a", title: "A", tags: ["Theme"], userTags: [] }, { id: "b", title: "B", tags: ["Other"], userTags: ["Manual"] }] as unknown as Movie[]
 const patch = vi.fn((id, tags) => { movies.find((movie) => movie.id === id)!.userTags = tags })
 const service = createMockTopicServices(() => movies, patch)
 const job = await service.startTagOrganization("selected", ["a"])
 expect(job.total).toBe(1)
 expect(patch).toHaveBeenCalledTimes(1)
 expect(patch).toHaveBeenCalledWith("a", ["Theme"])
 expect(movies[1]!.userTags).toEqual(["Manual"])
})
