import { afterEach, describe, expect, it } from "vitest"
import type { Movie } from "@/domain/movie/types"
import {
  findPlaylistIndex,
  listPlayerPlaylistMovies,
  readPlaylistAutoAdvance,
  recenterPlaylistWindow,
  resolvePlayerPlaylistSource,
  slidePlaylistWindow,
  writePlaylistAutoAdvance,
  PLAYER_PLAYLIST_AUTO_ADVANCE_STORAGE_KEY,
  PLAYER_PLAYLIST_WINDOW_MAX,
  PLAYER_PLAYLIST_WINDOW_RADIUS,
} from "@/lib/player-playlist"

function movie(id: string, overrides: Partial<Movie> = {}): Movie {
  return {
    id,
    title: id,
    code: id,
    studio: "Studio",
    actors: [],
    tags: [],
    userTags: [],
    runtimeMinutes: 100,
    rating: 4,
    summary: "",
    isFavorite: false,
    addedAt: `2026-07-${id.padStart(2, "0")}T00:00:00Z`,
    location: `D:/${id}.mp4`,
    resolution: "1080p",
    year: 2026,
    tone: "",
    coverClass: "",
    ...overrides,
  }
}

describe("resolvePlayerPlaylistSource", () => {
  it("follows the browsing origin through the movie detail hop", () => {
    expect(resolvePlayerPlaylistSource({ back: "browse" })).toBe("browse")
    expect(resolvePlayerPlaylistSource({ back: "actor" })).toBe("actor")
    expect(resolvePlayerPlaylistSource({ back: "detail", browse: "library" })).toBe("browse")
    expect(
      resolvePlayerPlaylistSource({
        actor: "Mina",
        back: "detail",
        browse: "library",
        detailBack: "actor",
      }),
    ).toBe("actor")
    expect(
      resolvePlayerPlaylistSource({
        back: "detail",
        browse: "library",
        detailBack: "home",
      }),
    ).toBeNull()
    expect(resolvePlayerPlaylistSource({ back: "home" })).toBeNull()
    expect(resolvePlayerPlaylistSource({ back: "history" })).toBeNull()
    expect(resolvePlayerPlaylistSource({ back: "curated-frames" })).toBeNull()
    expect(resolvePlayerPlaylistSource({ browse: "library" })).toBeNull()
  })
})

describe("listPlayerPlaylistMovies", () => {
  const movies = [
    movie("a", { actors: ["Mina"], addedAt: "2026-08-03T00:00:00Z" }),
    movie("b", { actors: ["Other"], addedAt: "2026-08-02T00:00:00Z" }),
    movie("c", { actors: ["Mina"], addedAt: "2026-08-01T00:00:00Z", isFavorite: true }),
  ]

  it("builds the actor page queue in cache order", () => {
    expect(
      listPlayerPlaylistMovies({
        source: "actor",
        movies,
        trashedMovies: [],
        query: { back: "actor", actor: "Mina" },
        hasPlayedMovie: () => false,
      }).map((item) => item.id),
    ).toEqual(["a", "c"])
  })

  it("builds the library queue with the current sort", () => {
    expect(
      listPlayerPlaylistMovies({
        source: "browse",
        movies,
        trashedMovies: [],
        query: { back: "browse", browse: "library" },
        hasPlayedMovie: () => false,
      }).map((item) => item.id),
    ).toEqual(["a", "b", "c"])
  })

  it("builds the library queue after library -> detail -> player", () => {
    expect(
      listPlayerPlaylistMovies({
        source: resolvePlayerPlaylistSource({ back: "detail", browse: "library" }),
        movies,
        trashedMovies: [],
        query: { back: "detail", browse: "library" },
        hasPlayedMovie: () => false,
      }).map((item) => item.id),
    ).toEqual(["a", "b", "c"])
  })

  it("returns nothing without a playlist source", () => {
    expect(
      listPlayerPlaylistMovies({
        source: null,
        movies,
        trashedMovies: [],
        query: { back: "detail", browse: "library", detailBack: "home" },
        hasPlayedMovie: () => false,
      }),
    ).toEqual([])
  })
})

describe("playlist window", () => {
  it("centers the initial window on the current item", () => {
    expect(recenterPlaylistWindow(12, 40)).toEqual({
      start: 12 - PLAYER_PLAYLIST_WINDOW_RADIUS,
      end: 12 + PLAYER_PLAYLIST_WINDOW_RADIUS,
    })
    expect(recenterPlaylistWindow(0, 8)).toEqual({ start: 0, end: 7 })
    expect(findPlaylistIndex([movie("a"), movie("b")], "b")).toBe(1)
    expect(findPlaylistIndex([movie("a")], "missing")).toBe(-1)
  })

  it("slides toward the scrolled edge without dropping the current item", () => {
    const window = { start: 10, end: 30 }
    expect(slidePlaylistWindow(window, "up", 20, 80).start).toBe(6)
    expect(slidePlaylistWindow(window, "down", 20, 80).end).toBe(34)
    const nearStart = slidePlaylistWindow({ start: 0, end: 20 }, "up", 4, 80)
    expect(nearStart.start).toBe(0)
    expect(nearStart.end).toBeGreaterThanOrEqual(4)
  })

  it("keeps sliding past the current item so later queue entries can appear", () => {
    let window = { start: 0, end: 10 }
    for (let step = 0; step < 12; step += 1) {
      window = slidePlaylistWindow(window, "down", 0, 80)
    }
    expect(window.end).toBeGreaterThan(24)
    expect(window.end - window.start + 1).toBeLessThanOrEqual(PLAYER_PLAYLIST_WINDOW_MAX)
    expect(window.start).toBeGreaterThan(0)
  })
})

describe("playlist auto-advance persistence", () => {
  afterEach(() => {
    localStorage.removeItem(PLAYER_PLAYLIST_AUTO_ADVANCE_STORAGE_KEY)
  })

  it("defaults to on and remembers an explicit off value", () => {
    expect(readPlaylistAutoAdvance()).toBe(true)
    writePlaylistAutoAdvance(false)
    expect(localStorage.getItem(PLAYER_PLAYLIST_AUTO_ADVANCE_STORAGE_KEY)).toBe("0")
    expect(readPlaylistAutoAdvance()).toBe(false)
    writePlaylistAutoAdvance(true)
    expect(readPlaylistAutoAdvance()).toBe(true)
  })
})
