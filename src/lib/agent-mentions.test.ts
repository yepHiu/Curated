import { describe, expect, it } from "vitest"
import type { Movie } from "@/domain/movie/types"
import {
  applyMention,
  mentionPickerLimit,
  mentionQueryAtCursor,
  searchActorMentions,
  searchComicMentions,
  searchMovieMentions,
  searchPhotoMentions,
  searchTagMentions,
} from "./agent-mentions"

function movie(partial: Partial<Movie> & { id: string; title: string }): Movie {
  return {
    code: "",
    studio: "",
    actors: [],
    tags: [],
    userTags: [],
    runtimeMinutes: 0,
    rating: 0,
    summary: "",
    isFavorite: false,
    addedAt: "",
    location: "",
    resolution: "",
    year: 0,
    tone: "",
    coverClass: "",
    ...partial,
  }
}

describe("agent mentions", () => {
  it("detects an @ query at the cursor", () => {
    expect(mentionQueryAtCursor("看看 @三", 5)).toEqual({ start: 3, query: "三" })
    expect(mentionQueryAtCursor("看看 三", 4)).toBeNull()
  })

  it("inserts a mention token", () => {
    const next = applyMention("看看 @三", 5, { kind: "actor", id: "三上", label: "三上" })
    expect(next.text).toBe("看看 @三上 ")
  })

  it("keeps a bare @ picker compact", () => {
    expect(mentionPickerLimit("")).toBe(2)
    expect(mentionPickerLimit("  ")).toBe(2)
    expect(mentionPickerLimit("三")).toBe(4)
  })

  it("filters movies and tags from the library cache", () => {
    const movies = [
      movie({ id: "m1", title: "Hello", code: "ABC-001", actors: ["Ada"], tags: ["轻松"], userTags: ["短片"] }),
      movie({ id: "m2", title: "Other", tags: ["剧情"] }),
    ]
    expect(searchMovieMentions(movies, "abc").map((item) => item.id)).toEqual(["m1"])
    expect(searchTagMentions(movies, "短").map((item) => item.label)).toEqual(["短片"])
    expect(searchActorMentions([{ name: "Ada" }, { name: "Bea" }], "ad").map((item) => item.id)).toEqual(["Ada"])
  })

  it("filters loaded comics and photo books for mentions", () => {
    expect(searchComicMentions([{ id: "c1", title: "Summer", tags: ["恋爱"] }, { id: "c2", title: "Other" }], "summer").map((item) => item.id)).toEqual(["c1"])
    expect(searchPhotoMentions([{ id: "p1", title: "Studio Book" }, { id: "p2", title: "Other" }], "studio").map((item) => item.kind)).toEqual(["photo"])
  })
})
