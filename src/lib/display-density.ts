export const RETINA_DESKTOP_DENSITY_QUERY =
  "(hover: hover) and (pointer: fine) and (min-width: 1024px) and (min-resolution: 2dppx)"

export interface MovieGridDensity {
  minTrackWidth: string
  cardMaxWidth: string
  gap: string
  minTrackPx: number
  gapPxEstimate: number
}

const DEFAULT_MOVIE_GRID_DENSITY: MovieGridDensity = {
  minTrackWidth: "var(--movie-grid-min-track)",
  cardMaxWidth: "min(100%, var(--movie-card-max-width))",
  gap: "var(--movie-grid-gap)",
  minTrackPx: 188,
  gapPxEstimate: 20,
}

const RETINA_COMPACT_MOVIE_GRID_DENSITY: MovieGridDensity = {
  minTrackWidth: "var(--movie-grid-min-track)",
  cardMaxWidth: "min(100%, var(--movie-card-max-width))",
  gap: "var(--movie-grid-gap)",
  minTrackPx: 172,
  gapPxEstimate: 16,
}

export function resolveMovieGridDensity(retinaDesktopCompact: boolean): MovieGridDensity {
  return retinaDesktopCompact
    ? RETINA_COMPACT_MOVIE_GRID_DENSITY
    : DEFAULT_MOVIE_GRID_DENSITY
}
