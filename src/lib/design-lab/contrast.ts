function normalizeHex(input: string): string {
  const value = input.trim().replace(/^#/, "")
  if (value.length === 3) {
    return value
      .split("")
      .map((char) => `${char}${char}`)
      .join("")
      .toLowerCase()
  }
  return value.toLowerCase()
}

function hexToRgb(hex: string): [number, number, number] {
  const normalized = normalizeHex(hex)
  if (!/^[0-9a-f]{6}$/i.test(normalized)) {
    throw new Error(`Unsupported hex color: ${hex}`)
  }

  return [
    Number.parseInt(normalized.slice(0, 2), 16),
    Number.parseInt(normalized.slice(2, 4), 16),
    Number.parseInt(normalized.slice(4, 6), 16),
  ]
}

function toLinearChannel(channel: number): number {
  const normalized = channel / 255
  if (normalized <= 0.03928) {
    return normalized / 12.92
  }
  return ((normalized + 0.055) / 1.055) ** 2.4
}

function getRelativeLuminance(hex: string): number {
  const [red, green, blue] = hexToRgb(hex)
  return (
    0.2126 * toLinearChannel(red) +
    0.7152 * toLinearChannel(green) +
    0.0722 * toLinearChannel(blue)
  )
}

export function getContrastRatio(foregroundHex: string, backgroundHex: string): number {
  const foregroundLuminance = getRelativeLuminance(foregroundHex)
  const backgroundLuminance = getRelativeLuminance(backgroundHex)
  const lighter = Math.max(foregroundLuminance, backgroundLuminance)
  const darker = Math.min(foregroundLuminance, backgroundLuminance)

  return Number(((lighter + 0.05) / (darker + 0.05)).toFixed(2))
}
