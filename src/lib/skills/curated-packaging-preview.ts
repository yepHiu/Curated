import type {
  PackagingBaseChange,
  PackagingMode,
} from "./curated-packaging-intent"

type VersionParts = {
  major: number
  minor: number
  patch: number
}

const parseVersion = (value: string): VersionParts => {
  const segments = value.split(".")
  if (segments.length !== 3) {
    throw new Error(`invalid version: ${value}`)
  }

  const [major, minor, patch] = segments.map((segment) => {
    const parsed = Number(segment)
    if (Number.isNaN(parsed)) {
      throw new Error(`invalid version: ${value}`)
    }
    return parsed
  })

  return { major, minor, patch }
}

const formatVersion = (parts: VersionParts): string =>
  `${parts.major}.${parts.minor}.${parts.patch}`

const applyBaseChange = (
  baseVersion: VersionParts,
  change: PackagingBaseChange | null
): VersionParts => {
  if (!change) {
    return baseVersion
  }

  const adjusted: VersionParts = { ...baseVersion, patch: 0 }

  if (change.major !== null) {
    adjusted.major = change.major
    adjusted.minor = 0
  }

  if (change.minor !== null) {
    adjusted.minor = change.minor
  }

  return adjusted
}

const shouldBumpPatch = (mode: PackagingMode): boolean =>
  mode !== "preview" && mode !== "set-base"

export type PackagingPreviewInput = {
  mode: PackagingMode
  currentBaseVersion: string
  baseChange: PackagingBaseChange | null
}

export type PackagingPreview = {
  mode: PackagingMode
  currentBaseVersion: string
  baseVersionAfterChange: string
  predictedVersion: string
  willBumpPatch: boolean
}

export const buildPackagingPreview = (
  input: PackagingPreviewInput
): PackagingPreview => {
  const parsed = parseVersion(input.currentBaseVersion)
  const baseAfterChange = applyBaseChange(parsed, input.baseChange)
  const willBumpPatch = shouldBumpPatch(input.mode)

  const predictedPatch = willBumpPatch
    ? baseAfterChange.patch + 1
    : baseAfterChange.patch

  return {
    mode: input.mode,
    currentBaseVersion: input.currentBaseVersion,
    baseVersionAfterChange: formatVersion(baseAfterChange),
    predictedVersion: formatVersion({
      ...baseAfterChange,
      patch: predictedPatch,
    }),
    willBumpPatch,
  }
}
