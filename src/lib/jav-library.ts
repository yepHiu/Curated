export type AppPage =
  | "library"
  | "favorites"
  | "recent"
  | "tags"
  | "detail"
  | "player"
  | "settings"

export type LibraryMode = Extract<AppPage, "library" | "favorites" | "recent" | "tags">
export type LibraryTab = "all" | "new" | "favorites" | "top-rated"

export interface Movie {
  id: string
  title: string
  code: string
  studio: string
  actors: string[]
  tags: string[]
  runtimeMinutes: number
  rating: number
  summary: string
  isFavorite: boolean
  addedAt: string
  location: string
  resolution: string
  year: number
  tone: string
  coverClass: string
}

export interface LibraryStat {
  label: string
  value: string
  detail: string
}

export interface LibrarySetting {
  id: string
  path: string
  title: string
}

export const libraryStats: LibraryStat[] = [
  {
    label: "Movies Indexed",
    value: "2,184",
    detail: "Across local and removable libraries",
  },
  {
    label: "Favorite Picks",
    value: "246",
    detail: "Curated for quick rewatch sessions",
  },
  {
    label: "Metadata Health",
    value: "98%",
    detail: "Poster, cast, and tags fully matched",
  },
]

export const libraryPaths: LibrarySetting[] = [
  {
    id: "library-a",
    path: "D:/Media/JAV/Main",
    title: "Primary archive",
  },
  {
    id: "library-b",
    path: "E:/Vault/JAV/New",
    title: "Recently imported",
  },
  {
    id: "library-c",
    path: "F:/Offline/Collections",
    title: "Cold storage",
  },
]

const movieSeeds: Omit<Movie, "id" | "code" | "location" | "addedAt">[] = [
  {
    title: "Midnight Kiss Broadcast",
    studio: "Velvet North",
    actors: ["Mina Kaze", "Rin Asuka"],
    tags: ["Romance", "4K", "Late Night"],
    runtimeMinutes: 134,
    rating: 4.8,
    summary:
      "A polished late-night feature with a slow-burn mood, crisp lighting, and strong cast chemistry.",
    isFavorite: true,
    resolution: "2160p",
    year: 2025,
    tone: "from-primary/35 via-primary/10 to-card",
    coverClass: "aspect-[4/5.6]",
  },
  {
    title: "Silk Line Directive",
    studio: "Studio Garnet",
    actors: ["Airi Sena"],
    tags: ["Drama", "Office", "High Rating"],
    runtimeMinutes: 126,
    rating: 4.7,
    summary:
      "An elegant office-set release with high production values and detailed metadata coverage.",
    isFavorite: true,
    resolution: "1080p",
    year: 2025,
    tone: "from-secondary via-accent/60 to-card",
    coverClass: "aspect-[4/4.8]",
  },
  {
    title: "Neon Velvet Archive",
    studio: "Moonlight Works",
    actors: ["Yua Mori", "Nao Shin"],
    tags: ["Sci-Fi", "Stylized", "New"],
    runtimeMinutes: 118,
    rating: 4.5,
    summary:
      "A stylized catalog favorite that mixes strong visual direction with a fast pace.",
    isFavorite: false,
    resolution: "2160p",
    year: 2026,
    tone: "from-accent via-primary/15 to-card",
    coverClass: "aspect-[4/5.2]",
  },
  {
    title: "Horizon Zero Kisses",
    studio: "North Pier",
    actors: ["Emi Kisaragi"],
    tags: ["Travel", "Outdoor", "Recently Added"],
    runtimeMinutes: 142,
    rating: 4.2,
    summary:
      "A travel-heavy feature with standout scenery and a well-tagged scene structure.",
    isFavorite: false,
    resolution: "1080p",
    year: 2026,
    tone: "from-muted via-primary/10 to-card",
    coverClass: "aspect-[4/5.8]",
  },
  {
    title: "Private Room Memoir",
    studio: "Golden Frame",
    actors: ["Sora Minami", "Miu Arata"],
    tags: ["Character", "Favorites", "Longform"],
    runtimeMinutes: 151,
    rating: 4.9,
    summary:
      "One of the strongest longform entries in the library, with rich cast notes and clean artwork.",
    isFavorite: true,
    resolution: "2160p",
    year: 2024,
    tone: "from-primary/25 via-accent/50 to-card",
    coverClass: "aspect-[4/5]",
  },
  {
    title: "Lovers in Static",
    studio: "Afterglow",
    actors: ["Kanna Rei"],
    tags: ["Moody", "Slow Burn", "Archive"],
    runtimeMinutes: 129,
    rating: 4.1,
    summary:
      "A moody catalog entry used as a reference for poster-heavy browsing and tag grouping.",
    isFavorite: false,
    resolution: "1080p",
    year: 2023,
    tone: "from-secondary/80 via-muted to-card",
    coverClass: "aspect-[4/4.6]",
  },
]

const codePrefixes = ["MKB", "SLD", "NVA", "HZK", "PRM", "LVS", "KTR", "AMR", "VLT", "NOA"]
const storagePools = [
  "D:/Media/JAV/Main",
  "E:/Vault/JAV/New",
  "F:/Offline/Collections",
]

const buildMovie = (index: number): Movie => {
  const seed = movieSeeds[index % movieSeeds.length]
  const prefix = codePrefixes[index % codePrefixes.length]
  const serial = String(100 + index).padStart(3, "0")
  const month = String((index % 12) + 1).padStart(2, "0")
  const day = String((index % 27) + 1).padStart(2, "0")
  const runtimeOffset = index % 17
  const ratingOffset = (index % 5) * 0.1
  const yearOffset = index % 3
  const storage = storagePools[index % storagePools.length]

  return {
    ...seed,
    id: `${prefix.toLowerCase()}-${serial}`,
    title: `${seed.title} ${index + 1}`,
    code: `${prefix}-${serial}`,
    runtimeMinutes: seed.runtimeMinutes + runtimeOffset,
    rating: Math.max(3.9, Number((seed.rating - ratingOffset).toFixed(1))),
    isFavorite: index % 4 === 0 ? true : seed.isFavorite,
    addedAt: `2026-${month}-${day}`,
    location: `${storage}/${prefix}-${serial}.${index % 2 === 0 ? "mkv" : "mp4"}`,
    year: seed.year + yearOffset,
    tags: [...seed.tags, index % 6 === 0 ? "Trending" : "Catalog"],
  }
}

export const movies: Movie[] = Array.from({ length: 180 }, (_, index) => buildMovie(index))

export const scanIntervals = [
  { label: "Every 30 minutes", value: "1800" },
  { label: "Every hour", value: "3600" },
  { label: "Every 6 hours", value: "21600" },
  { label: "Daily", value: "86400" },
]

export const formatRuntime = (minutes: number) => {
  const hours = Math.floor(minutes / 60)
  const remainder = minutes % 60

  return `${hours}h ${remainder}m`
}

export const formatAddedDate = (value: string) =>
  new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    year: "numeric",
  }).format(new Date(value))

export const getMovieById = (movieId?: string) =>
  movies.find((movie) => movie.id === movieId) ?? movies[0]

export const getRelatedMovies = (movieId: string, limit = 6) =>
  movies.filter((movie) => movie.id !== movieId).slice(0, limit)

export const getLibraryModeTitle = (mode: LibraryMode) => {
  switch (mode) {
    case "favorites":
      return {
        title: "Favorite shelf",
        description:
          "A hand-picked collection for quick access to your best tagged titles and most polished metadata entries.",
      }
    case "recent":
      return {
        title: "Recent imports",
        description:
          "Freshly scanned files, recently scraped posters, and newly matched cast details in one focused queue.",
      }
    case "tags":
      return {
        title: "Tag explorer",
        description:
          "Browse the library by mood, category, and metadata clusters without leaving the main shell.",
      }
    default:
      return {
        title: "Discover your catalog",
        description:
          "A music-style browsing surface adapted for movies, with fast search, category tabs, and detail previews.",
      }
  }
}
