/** Catalog-code helpers mirrored from backend scanner.ExtractNumber and moviecode.Classify. */

const sitePrefixPattern = /^[a-z0-9]+\.[a-z]{2,4}@/i
const suffixCleanPattern = /[-_]+(C|UC|U|HD|FHD|4K|uncensored|censored|leak|leaked)$/i

const numberPatterns: Array<{
  re: RegExp
  format: (matches: RegExpMatchArray) => string
}> = [
  {
    re: /\b(?:FC2[-_ ]?(?:PPV[-_ ]?)?|fc2)(\d{5,7})\b/i,
    format: (m) => `FC2-${m[1]}`,
  },
  {
    re: /\b(HEYZO)[-_ ]?(\d{3,6})\b/i,
    format: (m) => `HEYZO-${m[2]}`,
  },
  {
    re: /\b(TOKYO[-_ ]?HOT)[-_ ]?([a-z0-9-]+)\b/i,
    format: (m) => `TOKYO-HOT-${m[2]!.toUpperCase()}`,
  },
  {
    re: /\b(1PONDO|1PON)[-_ ]?(\d{6,10}[-_ ]?\d{0,4})\b/i,
    format: (m) => `1PONDO-${m[2]!.replace(/ /g, "")}`,
  },
  {
    re: /\b(CARIBBEANCOM|CARIB)[-_ ]?(\d{6,10}[-_ ]?\d{0,4})\b/i,
    format: (m) => `CARIBBEANCOM-${m[2]!.replace(/ /g, "")}`,
  },
  {
    re: /\b([a-z]{2,6})[-_ ]?(\d{2,5})\b/i,
    format: (m) => `${m[1]!.toUpperCase()}-${m[2]}`,
  },
]

export type MovieCodeMatchKind = "exact" | "similar"

export function normalizeMovieCode(value: string): string {
  return value.trim().toLowerCase().replace(/[ _]/g, "-")
}

export function fileBaseName(name: string): string {
  const normalized = name.replaceAll("\\", "/")
  const idx = normalized.lastIndexOf("/")
  return idx >= 0 ? normalized.slice(idx + 1) : normalized
}

export function cleanMovieFilename(rawName: string): string {
  const extIdx = rawName.lastIndexOf(".")
  let name = extIdx > 0 ? rawName.slice(0, extIdx) : rawName
  name = name.trim().replace(sitePrefixPattern, "")
  name = name.replace(suffixCleanPattern, "")
  return name.trim()
}

export function extractMovieNumber(filename: string): string {
  const cleaned = cleanMovieFilename(fileBaseName(filename))
  if (!cleaned) return ""
  for (const pattern of numberPatterns) {
    const matches = cleaned.match(pattern.re)
    if (matches && matches.length > 1) {
      return pattern.format(matches)
    }
  }
  return ""
}

export function classifyMovieCodes(incoming: string, existing: string): MovieCodeMatchKind | "" {
  const a = normalizeMovieCode(incoming)
  const b = normalizeMovieCode(existing)
  if (!a || !b) return ""
  if (a === b) return "exact"
  if (a.replaceAll("-", "") === b.replaceAll("-", "")) return "exact"
  if (a.startsWith(`${b}-`) || b.startsWith(`${a}-`)) return "similar"
  return ""
}

export function strongerMovieCodeMatch(
  left: MovieCodeMatchKind | "",
  right: MovieCodeMatchKind | "",
): MovieCodeMatchKind | "" {
  const rank = (kind: MovieCodeMatchKind | "") => (kind === "exact" ? 2 : kind === "similar" ? 1 : 0)
  return rank(left) >= rank(right) ? left : right
}
