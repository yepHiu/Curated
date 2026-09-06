export type FrameVisualHash = { id: string; movieId: string; bits: bigint }

// Difference hash is a review hint, never an automatic deletion decision.
export function frameDifferenceHash(rgba: Uint8ClampedArray): bigint | null {
  if (rgba.length !== 9 * 8 * 4) return null
  const luminance = Array.from({ length: 72 }, (_, i) => (rgba[i*4]! * 299 + rgba[i*4+1]! * 587 + rgba[i*4+2]! * 114) / 1000)
  if (Math.max(...luminance) - Math.min(...luminance) < 12) return null
  let bits = 0n
  for (let y=0; y<8; y++) for (let x=0; x<8; x++) bits = (bits << 1n) | (luminance[y*9+x]! > luminance[y*9+x+1]! ? 1n : 0n)
  return bits
}

export function findVisualFramePairs(hashes: readonly FrameVisualHash[], threshold = 6): [string,string][] {
  const pairs: [string,string][] = []
  for (let i=0;i<hashes.length;i++) for (let j=i+1;j<hashes.length;j++) {
    const a=hashes[i]!, b=hashes[j]!
    if (a.movieId !== b.movieId) continue
    let difference=a.bits^b.bits, distance=0
    while(difference) { difference &= difference-1n; distance++ }
    if (distance <= threshold) pairs.push([a.id,b.id])
    if (pairs.length >= 50) return pairs
  }
  return pairs
}

export async function hashFrameImage(url: string, signal: AbortSignal): Promise<bigint | null> {
  const response = await fetch(url, { signal, credentials:'include' })
  if (!response.ok) throw new Error('Image unavailable')
  const bitmap = await createImageBitmap(await response.blob())
  try {
    if (signal.aborted) return null
    const canvas=document.createElement('canvas');canvas.width=9;canvas.height=8
    const ctx=canvas.getContext('2d', { willReadFrequently:true });if(!ctx)return null
    ctx.drawImage(bitmap,0,0,9,8)
    return frameDifferenceHash(ctx.getImageData(0,0,9,8).data)
  } finally { bitmap.close() }
}
