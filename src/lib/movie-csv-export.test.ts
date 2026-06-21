import { describe, expect, it } from "vitest"
import type { Movie } from "@/domain/movie/types"
import {
  buildMovieCsv,
  buildMovieCsvBlob,
  buildMovieCsvFilename,
} from "./movie-csv-export"

function movie(overrides: Partial<Movie> = {}): Movie {
  return {
    id: "movie-1",
    title: "Example title",
    code: "ABC-123",
    studio: "Example Studio",
    actors: ["Alice", "Bella"],
    tags: [],
    userTags: [],
    runtimeMinutes: 120,
    rating: 0,
    summary: "",
    isFavorite: false,
    addedAt: "2026-01-01T00:00:00.000Z",
    location: "D:/Media/ABC-123.mp4",
    resolution: "1080p",
    year: 2026,
    tone: "",
    coverClass: "",
    ...overrides,
  }
}

describe("buildMovieCsv", () => {
  it("exports stable movie columns with actors joined in one cell", () => {
    const csv = buildMovieCsv([
      movie({
        code: "ABC-123",
        actors: ["Alice", "Bella"],
        title: "First title",
        releaseDate: "2026-01-02",
        studio: "Studio One",
      }),
    ])

    expect(csv).toBe(
      "\ufeff番号,演员,标题,影片发布日期,影片片商\r\nABC-123,Alice; Bella,First title,2026-01-02,Studio One",
    )
  })

  it("escapes commas quotes and newlines using CSV rules", () => {
    const csv = buildMovieCsv([
      movie({
        code: 'ABC"123',
        actors: ["Alice, A", 'Bella "B"'],
        title: "Line one\nLine two",
        releaseDate: "2026-01-02",
        studio: "Studio, One",
      }),
    ])

    expect(csv).toBe(
      '\ufeff番号,演员,标题,影片发布日期,影片片商\r\n"ABC""123","Alice, A; Bella ""B""","Line one\nLine two",2026-01-02,"Studio, One"',
    )
  })

  it("keeps missing values as empty cells", () => {
    const csv = buildMovieCsv([
      movie({
        actors: [],
        releaseDate: undefined,
        studio: "",
      }),
    ])

    expect(csv).toBe(
      "\ufeff番号,演员,标题,影片发布日期,影片片商\r\nABC-123,,Example title,,",
    )
  })

  it("prefixes spreadsheet formula-like text cells with a single quote", () => {
    const csv = buildMovieCsv([
      movie({
        code: "=ABC-123",
        actors: ["+Alice"],
        title: "-Title",
        releaseDate: "2026-01-02",
        studio: "@Studio",
      }),
    ])

    expect(csv).toBe(
      "\ufeff番号,演员,标题,影片发布日期,影片片商\r\n'=ABC-123,'+Alice,'-Title,2026-01-02,'@Studio",
    )
  })
})

describe("buildMovieCsvBlob", () => {
  it("creates a UTF-8 CSV blob", async () => {
    const blob = buildMovieCsvBlob([movie()])

    expect(blob.type).toBe("text/csv;charset=utf-8")
    await expect(blob.text()).resolves.toContain("番号,演员,标题,影片发布日期,影片片商")
  })
})

describe("buildMovieCsvFilename", () => {
  it("uses a stable local timestamp filename", () => {
    const filename = buildMovieCsvFilename(new Date("2026-06-21T09:08:07"))

    expect(filename).toBe("curated-movies-20260621-090807.csv")
  })
})
