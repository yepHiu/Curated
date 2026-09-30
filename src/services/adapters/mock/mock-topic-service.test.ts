// @vitest-environment jsdom
import { beforeEach, expect, it, vi } from "vitest"
import type { Movie } from "@/domain/movie/types"

beforeEach(() => { localStorage.clear(); vi.resetModules() })
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
