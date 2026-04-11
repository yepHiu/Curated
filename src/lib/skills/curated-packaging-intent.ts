export type PackagingMode = "publish" | "installer" | "portable" | "preview" | "set-base"

export type PackagingBaseChange = {
  major: number | null
  minor: number | null
}

export type PackagingIntent = {
  mode: PackagingMode
  baseChange: PackagingBaseChange | null
}

const packagingTriggers: Record<PackagingMode, string[]> = {
  publish: ["\u751f\u4ea7\u5305", "\u6574\u673a\u5305", "publish"],
  installer: ["\u5b89\u88c5\u5305", "installer"],
  portable: ["\u4fbf\u643a\u5305", "portable"],
  preview: ["\u9884\u89c8", "preview"],
  "set-base": [],
}

const parseBaseChangeValue = (input: string, keyword: string): number | null => {
  const regex = new RegExp(`${keyword}\\s*(?:\\u5347\\u5230|to)\\s*(\\d+)`)
  const match = input.match(regex)
  if (!match) {
    return null
  }
  const value = Number(match[1])
  return Number.isNaN(value) ? null : value
}

const escapeRegex = (value: string) =>
  value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")

const negationPatterns = [
  "\\u4e0d\\u8981\\u6253",
  "\\u4e0d\\u8981",
  "\\u522b\\u6253",
  "\\u4e0d\\u6253",
  "\\bnot\\b",
  "\\bno\\b",
]

const isTriggerNegated = (input: string, trigger: string): boolean => {
  const regex = new RegExp(
    `(?:${negationPatterns.join("|")})\\s*${escapeRegex(trigger)}`
  )
  return regex.test(input)
}

export const detectPackagingIntent = (input: string): PackagingIntent => {
  const normalized = input.toLowerCase()
  const majorChange = parseBaseChangeValue(normalized, "major")
  const minorChange = parseBaseChangeValue(normalized, "minor")
  const baseChange =
    majorChange !== null || minorChange !== null
      ? { major: majorChange, minor: minorChange }
      : null

  const matchesTrigger = (keys: string[]): boolean =>
    keys.some((trigger) => {
      const normalizedTrigger = trigger.toLowerCase()
      return (
        normalized.includes(normalizedTrigger) &&
        !isTriggerNegated(normalized, normalizedTrigger)
      )
    })

  if (matchesTrigger(packagingTriggers.preview)) {
    return { mode: "preview", baseChange }
  }

  if (matchesTrigger(packagingTriggers.portable)) {
    return { mode: "portable", baseChange }
  }

  if (matchesTrigger(packagingTriggers.installer)) {
    return { mode: "installer", baseChange }
  }

  if (matchesTrigger(packagingTriggers.publish)) {
    return { mode: "publish", baseChange }
  }

  if (baseChange) {
    return { mode: "set-base", baseChange }
  }

  return { mode: "preview", baseChange }
}
