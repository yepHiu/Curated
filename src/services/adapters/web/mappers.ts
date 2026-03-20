import type { MovieDetailDTO, MovieListItemDTO } from "@/api/types"
import type { Movie } from "@/domain/movie/types"

const TONE_POOL = [
  "from-primary/35 via-primary/10 to-card",
  "from-secondary via-accent/60 to-card",
  "from-accent via-primary/15 to-card",
  "from-muted via-primary/10 to-card",
  "from-primary/25 via-accent/50 to-card",
  "from-secondary/80 via-muted to-card",
]

const COVER_CLASS_POOL = [
  "aspect-[4/5.6]",
  "aspect-[4/4.8]",
  "aspect-[4/5.2]",
  "aspect-[4/5.8]",
  "aspect-[4/5]",
  "aspect-[4/4.6]",
]

function stableHash(id: string): number {
  let hash = 0
  for (let i = 0; i < id.length; i++) {
    hash = ((hash << 5) - hash + id.charCodeAt(i)) | 0
  }
  return Math.abs(hash)
}

export function mapMovieListItem(dto: MovieListItemDTO): Movie {
  const hash = stableHash(dto.id)
  return {
    id: dto.id,
    title: dto.title,
    code: dto.code,
    studio: dto.studio,
    actors: dto.actors ?? [],
    tags: dto.tags ?? [],
    runtimeMinutes: dto.runtimeMinutes,
    rating: dto.rating,
    isFavorite: dto.isFavorite,
    addedAt: dto.addedAt,
    location: dto.location,
    resolution: dto.resolution,
    year: dto.year,
    summary: "",
    tone: TONE_POOL[hash % TONE_POOL.length],
    coverClass: COVER_CLASS_POOL[hash % COVER_CLASS_POOL.length],
    coverUrl: dto.coverUrl,
    thumbUrl: dto.thumbUrl,
  }
}

export function mapMovieDetail(dto: MovieDetailDTO): Movie {
  const base = mapMovieListItem(dto)
  return {
    ...base,
    summary: dto.summary ?? "",
    previewImages: dto.previewImages ?? [],
    previewVideoUrl: dto.previewVideoUrl,
  }
}
