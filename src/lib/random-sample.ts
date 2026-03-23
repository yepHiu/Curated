/** 稳定字符串哈希，供种子随机使用 */
export function hashStringToUint32(s: string): number {
  let h = 0
  for (let i = 0; i < s.length; i++) {
    h = Math.imul(31, h) + s.charCodeAt(i) | 0
  }
  return h >>> 0
}

/** 小型快速 PRNG（mulberry32） */
export function mulberry32(seed: number): () => number {
  let a = seed >>> 0
  return () => {
    a = (a + 0x6d2b79f5) >>> 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = t + (Math.imul(t ^ (t >>> 7), 61 | t) ^ t)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

export function shuffleArraySeeded<T>(items: readonly T[], rng: () => number): T[] {
  const a = [...items]
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(rng() * (i + 1))
    const t = a[i]!
    a[i] = a[j]!
    a[j] = t
  }
  return a
}

/** 从列表中按种子洗牌后取前 limit 条；同一 seedString + 同一输入顺序结果稳定 */
export function sampleRandomMovies<T>(items: readonly T[], limit: number, seedString: string): T[] {
  if (limit <= 0 || items.length === 0) return []
  const rng = mulberry32(hashStringToUint32(seedString))
  const shuffled = shuffleArraySeeded(items, rng)
  return shuffled.slice(0, Math.min(limit, shuffled.length))
}
