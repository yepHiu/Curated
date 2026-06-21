import type { Movie } from "@/domain/movie/types"

const MOVIE_CSV_HEADERS = ["番号", "演员", "标题", "影片发布日期", "影片片商"] as const
const UTF8_BOM = "\ufeff"
const FORMULA_PREFIX_PATTERN = /^[=+\-@]/

function sanitizeCell(value: string): string {
  const protectedValue = FORMULA_PREFIX_PATTERN.test(value) ? `'${value}` : value
  if (/[",\r\n]/.test(protectedValue)) {
    return `"${protectedValue.replaceAll('"', '""')}"`
  }
  return protectedValue
}

function movieRow(movie: Movie): string[] {
  return [
    movie.code,
    movie.actors.join("; "),
    movie.title,
    movie.releaseDate ?? "",
    movie.studio,
  ]
}

function padDatePart(value: number): string {
  return String(value).padStart(2, "0")
}

export function buildMovieCsv(movies: readonly Movie[]): string {
  const rows = [
    MOVIE_CSV_HEADERS.join(","),
    ...movies.map((movie) => movieRow(movie).map(sanitizeCell).join(",")),
  ]
  return `${UTF8_BOM}${rows.join("\r\n")}`
}

export function buildMovieCsvBlob(movies: readonly Movie[]): Blob {
  return new Blob([buildMovieCsv(movies)], { type: "text/csv;charset=utf-8" })
}

export function buildMovieCsvFilename(now: Date = new Date()): string {
  const year = now.getFullYear()
  const month = padDatePart(now.getMonth() + 1)
  const day = padDatePart(now.getDate())
  const hours = padDatePart(now.getHours())
  const minutes = padDatePart(now.getMinutes())
  const seconds = padDatePart(now.getSeconds())
  return `curated-movies-${year}${month}${day}-${hours}${minutes}${seconds}.csv`
}
