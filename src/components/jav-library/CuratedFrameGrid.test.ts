import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import CuratedFrameGrid from "./CuratedFrameGrid.vue"

vi.mock("./CuratedFrameCard.vue", () => ({
  default: {
    name: "CuratedFrameCard",
    props: [
      "row",
      "imageUrl",
      "positionLabel",
      "batchMode",
      "selected",
      "nearDuplicate",
      "sectionActor",
    ],
    emits: ["toggleSelection", "open", "contextmenu"],
    template:
      "<article data-card><span>{{ row.title }}</span><button data-toggle @click=\"$emit('toggleSelection', row.id, sectionActor)\">toggle</button><button data-open @click=\"$emit('open', row, sectionActor)\">open</button><button data-menu @click=\"$emit('contextmenu', $event, row, sectionActor)\">menu</button></article>",
  },
}))

const rowA = {
  id: "frame-a",
  movieId: "movie-a",
  title: "Scene A",
  code: "ABC-001",
  actors: ["Alice"],
  positionSec: 65,
  capturedAt: "2026-05-01T00:00:00.000Z",
  tags: [],
}

const rowB = {
  ...rowA,
  id: "frame-b",
  title: "Scene B",
  positionSec: 90,
}

describe("CuratedFrameGrid", () => {
  it("renders cards with formatted positions and selection state", () => {
    const wrapper = mount(CuratedFrameGrid, {
      props: {
        items: [
          { row: rowA, url: "blob:a" },
          { row: rowB, url: "blob:b" },
        ],
        batchMode: true,
        selectedIds: ["frame-a"],
        nearDuplicateIds: ["frame-b"],
        sectionActor: "Alice",
      },
    })
    const cards = wrapper.findAllComponents({ name: "CuratedFrameCard" })

    expect(cards).toHaveLength(2)
    expect(cards[0]!.props("imageUrl")).toBe("blob:a")
    expect(cards[0]!.props("positionLabel")).toBe("01:05")
    expect(cards[0]!.props("selected")).toBe(true)
    expect(cards[0]!.props("nearDuplicate")).toBe(false)
    expect(cards[0]!.props("sectionActor")).toBe("Alice")
    expect(cards[1]!.props("selected")).toBe(false)
    expect(cards[1]!.props("nearDuplicate")).toBe(true)
  })

  it("forwards card events with the original grid item", async () => {
    const wrapper = mount(CuratedFrameGrid, {
      props: {
        items: [{ row: rowA, url: "blob:a" }],
        batchMode: true,
        selectedIds: [],
        nearDuplicateIds: [],
        sectionActor: "Alice",
      },
    })

    await wrapper.get("[data-toggle]").trigger("click")
    await wrapper.get("[data-open]").trigger("click")
    await wrapper.get("[data-menu]").trigger("click")

    expect(wrapper.emitted("toggleSelection")).toEqual([["frame-a", "Alice"]])
    expect(wrapper.emitted("open")?.[0]?.[0]).toEqual({ row: rowA, url: "blob:a" })
    expect(wrapper.emitted("open")?.[0]?.[1]).toBe("Alice")
    expect(wrapper.emitted("contextmenu")?.[0]?.[1]).toEqual({ row: rowA, url: "blob:a" })
    expect(wrapper.emitted("contextmenu")?.[0]?.[2]).toBe("Alice")
  })
})
