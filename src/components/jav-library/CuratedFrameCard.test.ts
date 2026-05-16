import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import CuratedFrameCard from "./CuratedFrameCard.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

const row = {
  id: "frame-a",
  movieId: "movie-a",
  title: "Scene A",
  code: "ABC-001",
  actors: ["Alice"],
  positionSec: 65,
  capturedAt: "2026-05-01T00:00:00.000Z",
  tags: [],
}

describe("CuratedFrameCard", () => {
  it("renders frame image, code metadata, selection state, and duplicate badge without the movie title", () => {
    const wrapper = mount(CuratedFrameCard, {
      props: {
        row,
        imageUrl: "blob:frame-a",
        positionLabel: "01:05",
        batchMode: true,
        selected: true,
        nearDuplicate: true,
        sectionActor: "Alice",
      },
    })

    expect(wrapper.get("img").attributes("src")).toBe("blob:frame-a")
    expect(wrapper.get("img").attributes("alt")).toBe("ABC-001")
    expect(wrapper.text()).not.toContain("Scene A")
    expect(wrapper.text()).toContain("ABC-001 · 01:05")
    expect(wrapper.text()).toContain("curated.duplicateReviewBadge")
    expect(wrapper.get("input[type='checkbox']").attributes("checked")).toBeDefined()
  })

  it("emits card open, context menu, and selection events with the optional actor scope", async () => {
    const wrapper = mount(CuratedFrameCard, {
      props: {
        row,
        imageUrl: "blob:frame-a",
        positionLabel: "01:05",
        batchMode: true,
        selected: false,
        nearDuplicate: false,
        sectionActor: "Alice",
      },
    })

    await wrapper.get("input[type='checkbox']").setValue(true)
    await wrapper.get("button").trigger("click")
    await wrapper.get("button").trigger("contextmenu")

    expect(wrapper.emitted("toggleSelection")).toEqual([["frame-a", "Alice"]])
    expect(wrapper.emitted("open")).toEqual([[row, "Alice"]])
    expect(wrapper.emitted("contextmenu")?.[0]?.[1]).toEqual(row)
    expect(wrapper.emitted("contextmenu")?.[0]?.[2]).toBe("Alice")
  })

  it("hides selection controls outside batch mode", () => {
    const wrapper = mount(CuratedFrameCard, {
      props: {
        row,
        imageUrl: "blob:frame-a",
        positionLabel: "01:05",
        batchMode: false,
        selected: false,
        nearDuplicate: false,
      },
    })

    expect(wrapper.find("input[type='checkbox']").exists()).toBe(false)
  })
})
