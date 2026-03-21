export interface Movie {
  id: string
  title: string
  code: string
  studio: string
  actors: string[]
  /** 元数据/刮削标签 */
  tags: string[]
  /** 用户本地标签 */
  userTags: string[]
  runtimeMinutes: number
  /** 列表/展示用有效评分（用户分优先，否则站点评分） */
  rating: number
  /** 详情来自 API；列表项可能未带 */
  metadataRating?: number
  /** 用户评分；undefined 表示未加载或未设置，null 表示已清除覆盖 */
  userRating?: number | null
  summary: string
  isFavorite: boolean
  addedAt: string
  location: string
  resolution: string
  year: number
  /** 发行日 YYYY-MM-DD（来自元数据 / API） */
  releaseDate?: string
  tone: string
  coverClass: string
  /** 远程海报 URL（刮削元数据） */
  coverUrl?: string
  /** 远程缩略图 URL，列表卡片优先展示 */
  thumbUrl?: string
  /** 详情页样本图 / 预览图 */
  previewImages?: string[]
  previewVideoUrl?: string
}
