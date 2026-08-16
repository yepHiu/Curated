import { describe, expect, it } from "vitest"
import {
  clusterFrameMarkers,
  DEFAULT_FRAME_MARKER_MIN_SEPARATION_PX,
  type FrameMarkerInput,
} from "@/lib/player-frame-markers"

function markers(...positionsSec: number[]): FrameMarkerInput[] {
  return positionsSec.map((positionSec, index) => ({ id: `f${index}`, positionSec }))
}

describe("clusterFrameMarkers", () => {
  it("returns empty output for empty input or unusable duration", () => {
    expect(clusterFrameMarkers([], 100, 600)).toEqual([])
    expect(clusterFrameMarkers(markers(10), 0, 600)).toEqual([])
    expect(clusterFrameMarkers(markers(10), -5, 600)).toEqual([])
  })

  it("keeps well-separated markers unmerged with ratio and seek target on the frame", () => {
    const result = clusterFrameMarkers(markers(30, 300, 600), 600, 600)
    expect(result).toHaveLength(3)
    expect(result[0]).toEqual({ items: [{ id: "f0", positionSec: 30 }], ratio: 0.05, seekToSec: 30 })
    expect(result[1].ratio).toBeCloseTo(0.5)
    expect(result[2].ratio).toBeCloseTo(1)
    expect(result.every((cluster) => cluster.items.length === 1)).toBe(true)
  })

  it("merges markers whose pixel distance to the cluster anchor is under the threshold", () => {
    // 宽 600px / 时长 600s → 1s = 1px；默认阈值 12px，111 与簇首 100 差 11px 应并入
    const result = clusterFrameMarkers(markers(100, 103, 107, 111), 600, 600)
    expect(result).toHaveLength(1)
    expect(result[0].items).toHaveLength(4)
    expect(result[0].seekToSec).toBe(100)
    // 中点 = (100px + 111px) / 2 / 600px
    expect(result[0].ratio).toBeCloseTo(105.5 / 600)
  })

  it("starts a new cluster at the threshold boundary instead of chaining", () => {
    // 首帧 100s；112s 与首帧相差 12px（不小于阈值）应开新簇，
    // 且 113s 与新簇首 112s 相差 1px 应并入新簇
    const result = clusterFrameMarkers(markers(100, 112, 113), 600, 600)
    expect(result).toHaveLength(2)
    expect(result[0].items.map((item) => item.id)).toEqual(["f0"])
    expect(result[1].items.map((item) => item.id)).toEqual(["f1", "f2"])
    expect(result[1].seekToSec).toBe(112)
  })

  it("splits a long dense run into adjacent bounded clusters", () => {
    // 30 帧每帧相隔 2px，总跨度 60px > 12px：以 12px 窗口裂为多簇
    const input = Array.from({ length: 30 }, (_, index) => ({ id: `d${index}`, positionSec: 200 + index * 2 }))
    const result = clusterFrameMarkers(input, 600, 600)
    expect(result.length).toBeGreaterThan(1)
    // 每簇跨度严格小于 12px（簇首窗口基准）
    for (const cluster of result) {
      const first = cluster.items[0].positionSec
      const last = cluster.items[cluster.items.length - 1].positionSec
      expect(last - first).toBeLessThan(12)
      expect(cluster.seekToSec).toBe(first)
    }
    // 簇按时间升序首尾相接，无遗漏
    const flat = result.flatMap((cluster) => cluster.items)
    expect(flat).toHaveLength(30)
  })

  it("falls back to unmerged rendering before the first width measurement", () => {
    const result = clusterFrameMarkers(markers(100, 101, 102), 600, 0)
    expect(result).toHaveLength(3)
    expect(result.every((cluster) => cluster.items.length === 1)).toBe(true)
    expect(result[1].ratio).toBeCloseTo(101 / 600)
  })

  it("filters out-of-range and invalid positions", () => {
    const input: FrameMarkerInput[] = [
      { id: "neg", positionSec: -1 },
      { id: "over", positionSec: 601 },
      { id: "nan", positionSec: Number.NaN },
      { id: "inf", positionSec: Number.POSITIVE_INFINITY },
      { id: "ok", positionSec: 300 },
    ]
    const result = clusterFrameMarkers(input, 600, 600)
    expect(result).toHaveLength(1)
    expect(result[0].items[0].id).toBe("ok")
  })

  it("produces order-independent output for shuffled input", () => {
    const input: FrameMarkerInput[] = [
      { id: "a", positionSec: 50 },
      { id: "b", positionSec: 100 },
      { id: "c", positionSec: 103 },
      { id: "d", positionSec: 400 },
    ]
    const sorted = clusterFrameMarkers(input, 600, 600)
    const shuffled = clusterFrameMarkers([input[3], input[2], input[0], input[1]], 600, 600)
    expect(shuffled).toEqual(sorted)
  })

  it("honors a custom minSeparationPx option", () => {
    expect(DEFAULT_FRAME_MARKER_MIN_SEPARATION_PX).toBe(12)
    const tight = clusterFrameMarkers(markers(100, 112), 600, 600, { minSeparationPx: 20 })
    expect(tight).toHaveLength(1)
    const loose = clusterFrameMarkers(markers(100, 112), 600, 600, { minSeparationPx: 5 })
    expect(loose).toHaveLength(2)
  })
})
