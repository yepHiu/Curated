import { ref } from "vue"

/**
 * 图片版本号管理 - 用于强制刷新重新搜刮后的海报/缩略图
 * 由于浏览器会缓存图片 URL，重新搜刮后需要添加版本参数来强制刷新
 */

const imageVersions = ref<Map<string, number>>(new Map())

/**
 * 获取影片图片的版本号（用于图片 URL 查询参数）
 */
export function getMovieImageVersion(movieId: string): number {
  return imageVersions.value.get(movieId) ?? 0
}

/**
 * 递增影片图片版本号 - 在重新搜刮完成后调用
 */
export function bumpMovieImageVersion(movieId: string): void {
  const current = imageVersions.value.get(movieId) ?? 0
  imageVersions.value.set(movieId, current + 1)
}

/**
 * 构建带版本号的图片 URL
 */
export function buildVersionedImageUrl(url: string | undefined, version: number): string | undefined {
  if (!url) return undefined
  if (version === 0) return url

  const separator = url.includes("?") ? "&" : "?"
  return `${url}${separator}_v=${version}`
}
