import { describe, expect, it } from "vitest"
import type { RecommendationFeedbackDTO } from "@/api/types"
import {
  loadLocalRecommendationFeedback,
  RECOMMENDATION_FEEDBACK_STORAGE_KEY,
  saveLocalRecommendationFeedback,
} from "@/lib/recommendation-feedback-local-storage"

function feedback(overrides: Partial<RecommendationFeedbackDTO> = {}): RecommendationFeedbackDTO {
  return {
    id: "feedback_1",
    action: "less",
    targetType: "actor",
    targetValue: "Actor A",
    sourceMovieId: "m01",
    createdAt: "2026-07-21T00:00:00Z",
    updatedAt: "2026-07-21T00:00:00Z",
    ...overrides,
  }
}

describe("recommendation feedback localStorage", () => {
  it("round-trips active feedback and removes expired snoozes", () => {
    let raw: string | null = null
    const storage = {
      getItem: () => raw,
      setItem: (_key: string, value: string) => {
        raw = value
      },
    }
    saveLocalRecommendationFeedback([
      feedback(),
      feedback({
        id: "feedback_2",
        action: "snooze",
        targetType: "movie",
        targetValue: "m02",
        expiresAt: "2026-07-20T00:00:00Z",
      }),
    ], storage)
    expect(raw).toContain('"schemaVersion":1')
    expect(loadLocalRecommendationFeedback(storage, Date.parse("2026-07-21T00:00:00Z"))).toEqual([
      feedback(),
    ])
  })

  it("rejects corrupt schemas and duplicate semantic targets", () => {
    const read = (value: unknown) => loadLocalRecommendationFeedback({
      getItem: (key: string) => key === RECOMMENDATION_FEEDBACK_STORAGE_KEY ? JSON.stringify(value) : null,
    })
    expect(read({ schemaVersion: 2, items: [] })).toEqual([])
    expect(read({ schemaVersion: 1, items: [feedback(), feedback({ id: "feedback_2" })] })).toEqual([])
    expect(read({ schemaVersion: 1, items: [feedback(), feedback({ id: "feedback_1", targetValue: "Actor B" })] })).toEqual([])
    expect(read({
      schemaVersion: 1,
      items: [
        feedback({ targetValue: "Straße" }),
        feedback({ id: "feedback_2", targetValue: "STRASSE" }),
      ],
    })).toEqual([])
  })

  it("survives storage quota failures", () => {
    expect(() => saveLocalRecommendationFeedback([feedback()], {
      setItem: () => {
        throw new DOMException("quota", "QuotaExceededError")
      },
    })).not.toThrow()
  })
})
