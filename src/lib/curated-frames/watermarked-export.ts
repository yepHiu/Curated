import type { CuratedFrameDbRow } from "@/lib/curated-frames/db"

export type WatermarkedCuratedExportFormat = "jpg" | "webp" | "png"

export interface WatermarkedCuratedFrameSource {
  row: CuratedFrameDbRow
  /** Mock uses an object URL; Web API uses the frame image endpoint URL. */
  imageUrl: string
}

export interface WatermarkedExportTheme {
  background: string
  foreground: string
  mutedForeground: string
  primary: string
}

export interface WatermarkedBandLayout {
  bandHeight: number
  padding: number
  iconSize: number
  brandFontSize: number
  mainFontSize: number
  actorFontSize: number
}

const DEFAULT_THEME: WatermarkedExportTheme = {
  background: "#f4f6fc",
  foreground: "#0f1219",
  mutedForeground: "#5a6378",
  primary: "#fe628e",
}

const MIME_BY_FORMAT: Record<WatermarkedCuratedExportFormat, string> = {
  jpg: "image/jpeg",
  webp: "image/webp",
  png: "image/png",
}

const EXT_BY_FORMAT: Record<WatermarkedCuratedExportFormat, string> = {
  jpg: "jpg",
  webp: "webp",
  png: "png",
}

function cssToken(name: string, fallback: string): string {
  if (typeof document === "undefined") {
    return fallback
  }
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return value || fallback
}

export function currentWatermarkedExportTheme(): WatermarkedExportTheme {
  return {
    background: cssToken("--background", DEFAULT_THEME.background),
    foreground: cssToken("--foreground", DEFAULT_THEME.foreground),
    mutedForeground: cssToken("--muted-foreground", DEFAULT_THEME.mutedForeground),
    primary: cssToken("--primary", DEFAULT_THEME.primary),
  }
}

/**
 * Keeps the added card close to 10% of the source frame height while retaining
 * enough room for the brand lockup and the two metadata levels at small sizes.
 */
export function buildWatermarkedBandLayout(sourceHeight: number): WatermarkedBandLayout {
  const safeHeight = Math.max(1, Math.round(sourceHeight))
  const bandHeight = Math.max(64, Math.round(safeHeight * 0.1))
  const padding = Math.max(10, Math.round(bandHeight * 0.17))
  return {
    bandHeight,
    padding,
    iconSize: Math.max(20, Math.round(bandHeight * 0.24)),
    brandFontSize: Math.max(13, Math.round(bandHeight * 0.21)),
    mainFontSize: Math.max(13, Math.round(bandHeight * 0.19)),
    actorFontSize: Math.max(10, Math.round(bandHeight * 0.11)),
  }
}

function safeText(value: string | undefined): string {
  return value?.trim() ?? ""
}

function sanitizeFileSegment(value: string, fallback: string): string {
  const safe = value.trim().replace(/[/\\?%*:|"<>]/g, "_").slice(0, 80)
  return safe || fallback
}

export function buildWatermarkedFrameFilename(
  row: Pick<CuratedFrameDbRow, "code" | "positionSec" | "id">,
  format: WatermarkedCuratedExportFormat,
): string {
  const code = sanitizeFileSegment(row.code, "frame")
  const seconds = Number.isFinite(row.positionSec) && row.positionSec >= 0
    ? Math.floor(row.positionSec)
    : 0
  const id = sanitizeFileSegment(row.id, "x").slice(0, 8)
  return `curated-watermarked-${code}-${seconds}s-${id}.${EXT_BY_FORMAT[format]}`
}

function fitText(
  ctx: CanvasRenderingContext2D,
  value: string,
  maxWidth: number,
): string {
  if (maxWidth <= 0 || !value) return ""
  if (ctx.measureText(value).width <= maxWidth) return value
  const ellipsis = "…"
  let low = 0
  let high = value.length
  while (low < high) {
    const mid = Math.ceil((low + high) / 2)
    if (ctx.measureText(`${value.slice(0, mid)}${ellipsis}`).width <= maxWidth) {
      low = mid
    } else {
      high = mid - 1
    }
  }
  return low > 0 ? `${value.slice(0, low)}${ellipsis}` : ellipsis
}

function drawSparkles(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  size: number,
  color: string,
) {
  const center = size / 2
  const arm = size * 0.4
  const drawStar = (cx: number, cy: number, radius: number) => {
    ctx.beginPath()
    ctx.moveTo(cx, cy - radius)
    ctx.lineTo(cx - radius * 0.24, cy - radius * 0.24)
    ctx.lineTo(cx - radius, cy)
    ctx.lineTo(cx - radius * 0.24, cy + radius * 0.24)
    ctx.lineTo(cx, cy + radius)
    ctx.lineTo(cx + radius * 0.24, cy + radius * 0.24)
    ctx.lineTo(cx + radius, cy)
    ctx.lineTo(cx + radius * 0.24, cy - radius * 0.24)
    ctx.closePath()
    ctx.stroke()
  }

  ctx.save()
  ctx.translate(x, y)
  ctx.strokeStyle = color
  ctx.lineWidth = Math.max(1.5, size * 0.07)
  ctx.lineCap = "round"
  ctx.lineJoin = "round"
  drawStar(center, center, arm)
  ctx.beginPath()
  ctx.moveTo(size * 0.2, size * 0.1)
  ctx.lineTo(size * 0.2, size * 0.28)
  ctx.moveTo(size * 0.1, size * 0.19)
  ctx.lineTo(size * 0.3, size * 0.19)
  ctx.moveTo(size * 0.8, size * 0.72)
  ctx.lineTo(size * 0.8, size * 0.9)
  ctx.moveTo(size * 0.7, size * 0.81)
  ctx.lineTo(size * 0.9, size * 0.81)
  ctx.stroke()
  ctx.restore()
}

function drawWatermarkedFrame(
  image: CanvasImageSource,
  width: number,
  height: number,
  row: CuratedFrameDbRow,
  theme: WatermarkedExportTheme,
): HTMLCanvasElement {
  const layout = buildWatermarkedBandLayout(height)
  const canvas = document.createElement("canvas")
  canvas.width = width
  canvas.height = height + layout.bandHeight
  const ctx = canvas.getContext("2d")
  if (!ctx) {
    throw new Error("watermarked export canvas is unavailable")
  }

  ctx.drawImage(image, 0, 0, width, height)
  ctx.fillStyle = theme.background
  ctx.fillRect(0, height, width, layout.bandHeight)

  const leftX = layout.padding
  const brandCenterY = height + layout.bandHeight / 2
  drawSparkles(
    ctx,
    leftX,
    brandCenterY - layout.iconSize / 2,
    layout.iconSize,
    theme.primary,
  )
  ctx.fillStyle = theme.foreground
  ctx.font = `600 ${layout.brandFontSize}px Outfit, ui-sans-serif, system-ui, sans-serif`
  ctx.textBaseline = "middle"
  ctx.textAlign = "left"
  ctx.fillText("Curated", leftX + layout.iconSize + Math.max(8, layout.padding * 0.55), brandCenterY)

  const rightX = width - layout.padding
  const rightStart = Math.max(
    leftX + layout.iconSize + layout.padding * 3,
    Math.round(width * 0.36),
  )
  const rightMaxWidth = Math.max(0, rightX - rightStart)
  const code = safeText(row.code)
  const title = safeText(row.title) || "Untitled"
  const actors = row.actors.map(safeText).filter(Boolean).join(" · ")
  const gap = Math.max(6, Math.round(layout.mainFontSize * 0.4))

  ctx.font = `700 ${layout.mainFontSize}px Outfit, ui-sans-serif, system-ui, sans-serif`
  const codeWidth = code ? ctx.measureText(code).width : 0
  const titleMaxWidth = Math.max(0, rightMaxWidth - (code ? codeWidth + gap : 0))
  const fittedTitle = fitText(ctx, title, titleMaxWidth)
  const titleWidth = ctx.measureText(fittedTitle).width
  const codeX = rightX - titleWidth - (code ? gap : 0) - codeWidth
  const titleX = rightX - titleWidth
  const mainY = height + layout.padding + layout.mainFontSize
  ctx.textBaseline = "alphabetic"
  ctx.textAlign = "right"
  ctx.fillStyle = theme.foreground
  ctx.fillText(fittedTitle, titleX, mainY)
  if (code) {
    ctx.fillStyle = theme.primary
    ctx.textAlign = "left"
    ctx.fillText(code, codeX, mainY)
  }

  if (actors) {
    ctx.font = `500 ${layout.actorFontSize}px Outfit, ui-sans-serif, system-ui, sans-serif`
    ctx.fillStyle = theme.mutedForeground
    ctx.textAlign = "right"
    ctx.fillText(fitText(ctx, actors, rightMaxWidth), rightX, mainY + layout.actorFontSize + Math.max(3, layout.padding * 0.2))
  }

  return canvas
}

function loadImage(blob: Blob): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(blob)
    const image = new Image()
    image.onload = () => {
      URL.revokeObjectURL(url)
      resolve(image)
    }
    image.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error("failed to decode curated frame image"))
    }
    image.src = url
  })
}

async function sourceBlob(source: WatermarkedCuratedFrameSource): Promise<Blob> {
  if (source.row.imageBlob) {
    return source.row.imageBlob
  }
  const response = await fetch(source.imageUrl, { credentials: "include" })
  if (!response.ok) {
    throw new Error(`failed to load curated frame image (${response.status})`)
  }
  return response.blob()
}

function canvasToBlob(
  canvas: HTMLCanvasElement,
  format: WatermarkedCuratedExportFormat,
): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (!blob) {
          reject(new Error("failed to encode watermarked curated frame"))
          return
        }
        resolve(blob)
      },
      MIME_BY_FORMAT[format],
      format === "png" ? undefined : 0.9,
    )
  })
}

export async function renderWatermarkedCuratedFrame(
  source: WatermarkedCuratedFrameSource,
  format: WatermarkedCuratedExportFormat,
  theme: WatermarkedExportTheme = currentWatermarkedExportTheme(),
): Promise<Blob> {
  const image = await loadImage(await sourceBlob(source))
  if (!image.naturalWidth || !image.naturalHeight) {
    throw new Error("curated frame image has no dimensions")
  }
  const canvas = drawWatermarkedFrame(
    image,
    image.naturalWidth,
    image.naturalHeight,
    source.row,
    theme,
  )
  return canvasToBlob(canvas, format)
}

function crc32(data: Uint8Array): number {
  let crc = 0xffffffff
  for (const byte of data) {
    crc ^= byte
    for (let bit = 0; bit < 8; bit += 1) {
      crc = (crc >>> 1) ^ (crc & 1 ? 0xedb88320 : 0)
    }
  }
  return (crc ^ 0xffffffff) >>> 0
}

function writeU16(view: DataView, offset: number, value: number) {
  view.setUint16(offset, value, true)
}

function writeU32(view: DataView, offset: number, value: number) {
  view.setUint32(offset, value >>> 0, true)
}

/** Builds a standards-compliant, uncompressed ZIP for up to the existing 20-frame batch limit. */
export function buildStoredZip(entries: readonly { name: string; data: Uint8Array }[]): Blob {
  const encoder = new TextEncoder()
  const locals: Uint8Array[] = []
  const central: Uint8Array[] = []
  let offset = 0

  for (const entry of entries) {
    const name = encoder.encode(entry.name)
    const crc = crc32(entry.data)
    const local = new Uint8Array(30 + name.length + entry.data.length)
    const localView = new DataView(local.buffer)
    writeU32(localView, 0, 0x04034b50)
    writeU16(localView, 4, 20)
    writeU16(localView, 6, 0x0800)
    writeU16(localView, 8, 0)
    writeU16(localView, 10, 0)
    writeU16(localView, 12, 0)
    writeU32(localView, 14, crc)
    writeU32(localView, 18, entry.data.length)
    writeU32(localView, 22, entry.data.length)
    writeU16(localView, 26, name.length)
    writeU16(localView, 28, 0)
    local.set(name, 30)
    local.set(entry.data, 30 + name.length)
    locals.push(local)

    const directory = new Uint8Array(46 + name.length)
    const directoryView = new DataView(directory.buffer)
    writeU32(directoryView, 0, 0x02014b50)
    writeU16(directoryView, 4, 20)
    writeU16(directoryView, 6, 20)
    writeU16(directoryView, 8, 0x0800)
    writeU16(directoryView, 10, 0)
    writeU16(directoryView, 12, 0)
    writeU16(directoryView, 14, 0)
    writeU32(directoryView, 16, crc)
    writeU32(directoryView, 20, entry.data.length)
    writeU32(directoryView, 24, entry.data.length)
    writeU16(directoryView, 28, name.length)
    writeU16(directoryView, 30, 0)
    writeU16(directoryView, 32, 0)
    writeU16(directoryView, 34, 0)
    writeU16(directoryView, 36, 0)
    writeU32(directoryView, 38, 0)
    writeU32(directoryView, 42, offset)
    directory.set(name, 46)
    central.push(directory)
    offset += local.length
  }

  const centralOffset = offset
  const centralSize = central.reduce((total, item) => total + item.length, 0)
  const end = new Uint8Array(22)
  const endView = new DataView(end.buffer)
  writeU32(endView, 0, 0x06054b50)
  writeU16(endView, 4, 0)
  writeU16(endView, 6, 0)
  writeU16(endView, 8, entries.length)
  writeU16(endView, 10, entries.length)
  writeU32(endView, 12, centralSize)
  writeU32(endView, 16, centralOffset)
  writeU16(endView, 20, 0)

  const parts: BlobPart[] = [...locals, ...central, end].map((bytes) =>
    bytes.buffer instanceof ArrayBuffer
      ? bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)
      : new Uint8Array(bytes).buffer,
  )
  return new Blob(parts, { type: "application/zip" })
}

export async function buildWatermarkedCuratedFramesExport(
  sources: readonly WatermarkedCuratedFrameSource[],
  format: WatermarkedCuratedExportFormat,
  theme: WatermarkedExportTheme = currentWatermarkedExportTheme(),
): Promise<{ blob: Blob; filename: string }> {
  const rendered = await Promise.all(
    sources.map(async (source) => ({
      name: buildWatermarkedFrameFilename(source.row, format),
      data: new Uint8Array(await (await renderWatermarkedCuratedFrame(source, format, theme)).arrayBuffer()),
    })),
  )
  if (rendered.length === 1) {
    return {
      blob: new Blob([
        rendered[0]!.data.buffer instanceof ArrayBuffer
          ? rendered[0]!.data.buffer.slice(
              rendered[0]!.data.byteOffset,
              rendered[0]!.data.byteOffset + rendered[0]!.data.byteLength,
            )
          : new Uint8Array(rendered[0]!.data).buffer,
      ], { type: MIME_BY_FORMAT[format] }),
      filename: rendered[0]!.name,
    }
  }
  return {
    blob: buildStoredZip(rendered),
    filename: "curated-watermarked-frames-export.zip",
  }
}
