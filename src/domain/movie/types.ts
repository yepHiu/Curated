export interface Movie {
  id: string
  title: string
  code: string
  studio: string
  actors: string[]
  tags: string[]
  runtimeMinutes: number
  rating: number
  summary: string
  isFavorite: boolean
  addedAt: string
  location: string
  resolution: string
  year: number
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
