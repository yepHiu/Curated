export interface MovieGridChunkStyleOptions {
  minTrackWidth: string
  gap: string
}

/**
 * Build a CSS Grid auto-fill template that stays browser-valid.
 * The max track cannot be a nested `minmax(...)`; browsers drop the whole declaration.
 */
export function buildAutoFillMovieGridTemplate(minTrackWidth: string): string {
  return `repeat(auto-fill, minmax(min(100%, ${minTrackWidth}), 1fr))`
}

/**
 * Keep chunk spacing inside the measured grid element.
 * Virtual scroller item padding/gap outside the measured content is not reflected in item height.
 */
export function buildMovieGridChunkStyle(options: MovieGridChunkStyleOptions) {
  const { minTrackWidth, gap } = options

  return {
    gridTemplateColumns: buildAutoFillMovieGridTemplate(minTrackWidth),
    columnGap: gap,
    rowGap: gap,
    paddingBottom: gap,
  }
}
