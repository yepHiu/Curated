import { describe, expect, it } from "vitest"
import {
  resolveLibraryGridAction,
  resolveLibraryGridPageSelection,
  resolveLibraryGridSelection,
} from "@/lib/gamepad/library-grid-navigation"

const movies = Array.from({ length: 12 }, (_, index) => ({
  id: `movie-${index}`,
}))

describe("library grid gamepad navigation", () => {
  it("moves horizontally by one item", () => {
    expect(resolveLibraryGridSelection({
      movies,
      currentMovieId: "movie-0",
      direction: "right",
      columnCount: 5,
    })?.id).toBe("movie-1")
  })

  it("moves vertically by the current column count", () => {
    expect(resolveLibraryGridSelection({
      movies,
      currentMovieId: "movie-2",
      direction: "down",
      columnCount: 5,
    })?.id).toBe("movie-7")
  })

  it("clamps navigation to list boundaries", () => {
    expect(resolveLibraryGridSelection({
      movies,
      currentMovieId: "movie-0",
      direction: "left",
      columnCount: 5,
    })?.id).toBe("movie-0")

    expect(resolveLibraryGridSelection({
      movies,
      currentMovieId: "movie-11",
      direction: "down",
      columnCount: 5,
    })?.id).toBe("movie-11")
  })

  it("uses Square to enter batch mode before toggling existing batch selections", () => {
    expect(resolveLibraryGridAction({ button: "square", batchMode: false })).toBe("enter-batch-select")
    expect(resolveLibraryGridAction({ button: "square", batchMode: true })).toBe("toggle-batch-select")
  })

  it("uses page navigation for L2 and R2 jumps", () => {
    expect(resolveLibraryGridPageSelection({
      movies,
      currentMovieId: "movie-7",
      direction: "up",
      columnCount: 5,
      rowsPerPage: 4,
    })?.id).toBe("movie-0")

    expect(resolveLibraryGridPageSelection({
      movies,
      currentMovieId: "movie-2",
      direction: "down",
      columnCount: 5,
      rowsPerPage: 4,
    })?.id).toBe("movie-11")
  })
})
