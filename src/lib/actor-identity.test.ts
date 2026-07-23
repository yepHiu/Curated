import { describe, expect, it } from "vitest"
import { normalizeActorIdentity } from "@/lib/actor-identity"

describe("normalizeActorIdentity", () => {
  it("folds compatibility width and repeated whitespace", () => {
    expect(normalizeActorIdentity("  Ａlice\u3000  Smith  ")).toBe("alice smith")
  })

  it("uses comparison semantics equivalent to multi-code-point case folds", () => {
    expect(normalizeActorIdentity("Straße")).toBe(normalizeActorIdentity("STRASSE"))
  })

  it("unifies positional Greek sigma forms", () => {
    expect(normalizeActorIdentity("ΟΣ")).toBe(normalizeActorIdentity("ος"))
  })
})
