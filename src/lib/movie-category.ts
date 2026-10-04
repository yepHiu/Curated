import type { Movie } from "@/domain/movie/types"
import type { LibraryMode } from "@/domain/library/types"

/** 按番号识别 FC2，兼容 PPV、空格和连接符；用户标签不参与分类。 */
export function isFC2MovieCode(code: string | undefined): boolean {
  return (code ?? "").trim().toUpperCase().replace(/[\s_-]+/g, "").startsWith("FC2")
}

/** 浏览与播放列表共用分类范围，回收站保留完整的管理视图。 */
export function moviesInBrowseCategory(movies: readonly Movie[], mode: LibraryMode): Movie[] {
  return movies.filter((movie) => {
    // 回收站不隐藏另一类别中等待恢复或删除的作品。
    return mode === "trash" || isFC2MovieCode(movie.code) === (mode === "fc2")
  })
}
