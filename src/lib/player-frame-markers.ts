/** 播放器进度条萃取帧标记的密集合并（像素级贪心聚类，纯前端） */

export interface FrameMarkerInput {
  id: string
  positionSec: number
}

export interface FrameMarkerCluster {
  /** 簇内成员按时间升序 */
  items: FrameMarkerInput[]
  /** 展示位置（0–1），簇内首末成员像素坐标的中点 */
  ratio: number
  /** 点击跳转目标：簇内最早帧 */
  seekToSec: number
}

export interface FrameMarkerClusterOptions {
  /** 两个标记中心像素距离小于该值判定为密集；默认 12px（刻度视觉宽 + 间隙） */
  minSeparationPx?: number
}

export const DEFAULT_FRAME_MARKER_MIN_SEPARATION_PX = 12

/**
 * 把萃取帧时刻按进度条像素宽度做贪心分簇。
 *
 * - `durationSec <= 0` 时返回空（播放器进度条本就不可用）。
 * - `positionSec` 越界（< 0、> duration 或非法值）的脏数据被过滤。
 * - `trackWidthPx <= 0`（ResizeObserver 首测前）退化为不合并，每帧一簇。
 * - 分簇以簇首帧为窗口基准（而非链式比较）：新帧与簇首像素距离
 *   `< minSeparationPx` 才入簇，保证单簇宽度有界；更长的密集段
 *   自然裂为相邻多簇，各自带数量。
 */
export function clusterFrameMarkers(
  markers: readonly FrameMarkerInput[],
  durationSec: number,
  trackWidthPx: number,
  options?: FrameMarkerClusterOptions,
): FrameMarkerCluster[] {
  if (!(durationSec > 0)) return []

  const minSeparationPx = options?.minSeparationPx ?? DEFAULT_FRAME_MARKER_MIN_SEPARATION_PX
  const valid = markers
    .filter((marker) => Number.isFinite(marker.positionSec) && marker.positionSec >= 0 && marker.positionSec <= durationSec)
    .sort((a, b) => a.positionSec - b.positionSec)
  if (valid.length === 0) return []

  if (!(trackWidthPx > 0) || !(minSeparationPx > 0)) {
    return valid.map((marker) => ({
      items: [marker],
      ratio: marker.positionSec / durationSec,
      seekToSec: marker.positionSec,
    }))
  }

  const px = (positionSec: number) => (positionSec / durationSec) * trackWidthPx
  const groups: FrameMarkerInput[][] = []
  let current: FrameMarkerInput[] = [valid[0]]
  for (let index = 1; index < valid.length; index += 1) {
    const marker = valid[index]
    if (px(marker.positionSec) - px(current[0].positionSec) < minSeparationPx) {
      current.push(marker)
    } else {
      groups.push(current)
      current = [marker]
    }
  }
  groups.push(current)

  return groups.map((items) => {
    const first = items[0]
    const last = items[items.length - 1]
    const midpointPx = (px(first.positionSec) + px(last.positionSec)) / 2
    return {
      items,
      ratio: midpointPx / trackWidthPx,
      seekToSec: first.positionSec,
    }
  })
}
