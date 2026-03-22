/** 萃取帧元数据（Mock：图像 Blob 存 IndexedDB；Web API：图在后端库，前端仅 URL） */
export interface CuratedFrameRecord {
  id: string
  movieId: string
  title: string
  code: string
  actors: string[]
  positionSec: number
  capturedAt: string
  /** 仅萃取帧库使用，与影片 userTags / 元数据 tags 无关联 */
  tags: string[]
}

/** 列表展示用：含可展示的 object URL（由组件创建/回收） */
export interface CuratedFrameListItem extends CuratedFrameRecord {
  imageUrl: string
}

export type CuratedFrameSaveMode = "app" | "download" | "directory"
