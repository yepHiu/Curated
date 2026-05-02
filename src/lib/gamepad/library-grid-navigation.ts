import type {
  GamepadDirection,
  StandardGamepadButtonName,
} from "@/lib/gamepad/standard-gamepad"

export interface LibraryGridMovieRef {
  id: string
}

export interface ResolveLibraryGridSelectionOptions<T extends LibraryGridMovieRef> {
  movies: readonly T[]
  currentMovieId?: string | null
  direction: GamepadDirection
  columnCount: number
}

export type LibraryGridAction =
  | "open-details"
  | "enter-batch-select"
  | "toggle-batch-select"
  | "none"

export function resolveLibraryGridSelection<T extends LibraryGridMovieRef>(
  options: ResolveLibraryGridSelectionOptions<T>,
): T | null {
  const { movies, currentMovieId, direction } = options
  if (movies.length === 0) return null

  const columns = Math.max(1, Math.floor(options.columnCount))
  const currentIndex = Math.max(0, movies.findIndex((movie) => movie.id === currentMovieId))
  const delta =
    direction === "right" ? 1 :
    direction === "left" ? -1 :
    direction === "down" ? columns :
    -columns
  const nextIndex = Math.max(0, Math.min(movies.length - 1, currentIndex + delta))
  return movies[nextIndex] ?? null
}

export function resolveLibraryGridAction(options: {
  button: StandardGamepadButtonName
  batchMode: boolean
}): LibraryGridAction {
  if (options.button === "cross") return "open-details"
  if (options.button !== "square") return "none"
  return options.batchMode ? "toggle-batch-select" : "enter-batch-select"
}
