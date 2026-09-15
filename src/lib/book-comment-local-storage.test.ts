import { describe, expect, it } from "vitest"
import {
  MOCK_COMIC_COMMENTS_KEY,
  getLocalBookComment,
  putLocalBookComment,
  removeLocalBookComment,
} from "./book-comment-local-storage"

describe("book-comment-local-storage", () => {
  it("round-trips a note and removes it by id", () => {
    localStorage.clear()
    expect(getLocalBookComment(MOCK_COMIC_COMMENTS_KEY, "comic-1")).toEqual({
      body: "",
      updatedAt: "",
    })
    const saved = putLocalBookComment(MOCK_COMIC_COMMENTS_KEY, "comic-1", "hello")
    expect(saved.body).toBe("hello")
    expect(getLocalBookComment(MOCK_COMIC_COMMENTS_KEY, "comic-1").body).toBe("hello")
    removeLocalBookComment(MOCK_COMIC_COMMENTS_KEY, "comic-1")
    expect(getLocalBookComment(MOCK_COMIC_COMMENTS_KEY, "comic-1").body).toBe("")
  })
})
