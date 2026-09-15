import { afterEach, describe, expect, it } from "vitest"
import { buildBookVirtualChunks, parseBookGridColumnCount } from "./book-virtual-scroll"

describe("buildBookVirtualChunks", () => {
  it("fills complete rows except the last chunk", () => {
    const items = Array.from({ length: 10 }, (_, index) => ({ id: `b${index}` }))

    const chunks = buildBookVirtualChunks(items, 4, (item) => item.id)

    expect(chunks).toHaveLength(3)
    expect(chunks[0]?.items.map((item) => item.id)).toEqual(["b0", "b1", "b2", "b3"])
    expect(chunks[2]?.items.map((item) => item.id)).toEqual(["b8", "b9"])
    expect(chunks[0]?.sizeKey).toBe("b0|b1|b2|b3")
  })

  it("returns no chunks for an empty list", () => {
    expect(buildBookVirtualChunks([], 8, (item: { id: string }) => item.id)).toEqual([])
  })
})

describe("parseBookGridColumnCount", () => {
  const originalGetComputedStyle = window.getComputedStyle

  afterEach(() => {
    window.getComputedStyle = originalGetComputedStyle
  })

  it("counts explicit track list entries", () => {
    stubGridTemplateColumns("120px 120px 120px")
    expect(parseBookGridColumnCount(document.createElement("div"))).toBe(3)
  })
})

/** 用固定 grid-template-columns 覆盖 getComputedStyle，避免 jsdom 缺布局。 */
function stubGridTemplateColumns(value: string) {
  const original = window.getComputedStyle
  window.getComputedStyle = ((target: Element) => {
    const style = original.call(window, target)
    return new Proxy(style, {
      get(obj, prop) {
        if (prop === "gridTemplateColumns") return value
        const resolved = Reflect.get(obj, prop, obj)
        return typeof resolved === "function" ? resolved.bind(obj) : resolved
      },
    })
  }) as typeof window.getComputedStyle
}
