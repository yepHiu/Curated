/**
 * 阅读器相邻页原图预取：用隐藏 Image 解码，不把全书页挂进 DOM。
 */

/**
 * 预取给定原图 URL；返回取消函数，离开路由时应调用以丢掉引用。
 * 失败静默，不影响当前可见页。
 */
export function prefetchBookPageUrls(urls: readonly string[]): () => void {
  const images: HTMLImageElement[] = []
  const seen = new Set<string>()
  for (const raw of urls) {
    const url = raw.trim()
    if (!url || seen.has(url)) continue
    seen.add(url)
    const image = new Image()
    image.decoding = "async"
    image.src = url
    images.push(image)
  }
  return () => {
    for (const image of images) {
      image.src = ""
    }
  }
}

/**
 * 以可见页为中心收集 current±1 的原图 URL；stitch 时对每个可见页各扩一页。
 */
export function collectAdjacentBookPageUrls(
  pages: readonly { index?: number; imageUrl?: string }[],
  visibleIndexes: readonly number[],
): string[] {
  const wanted = new Set<number>()
  for (const index of visibleIndexes) {
    wanted.add(index - 1)
    wanted.add(index)
    wanted.add(index + 1)
  }
  const urls: string[] = []
  for (const page of pages) {
    const index = page.index
    if (index == null || !wanted.has(index)) continue
    const url = page.imageUrl?.trim()
    if (url) urls.push(url)
  }
  return urls
}
