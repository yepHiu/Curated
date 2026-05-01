import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import CuratedFrameActorsTab from "./CuratedFrameActorsTab.vue"

vi.mock("./CuratedFrameGrid.vue", () => ({
  default: {
    name: "CuratedFrameGrid",
    props: [
      "items",
      "batchMode",
      "selectedIds",
      "nearDuplicateIds",
      "sectionActor",
    ],
    emits: ["toggleSelection", "open", "contextmenu"],
    template:
      "<div data-grid><span>{{ sectionActor }}</span><button data-toggle @click=\"$emit('toggleSelection', items[0].row.id, sectionActor)\">toggle</button><button data-open @click=\"$emit('open', items[0], sectionActor)\">open</button><button data-menu @click=\"$emit('contextmenu', $event, items[0], sectionActor)\">menu</button></div>",
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
  actors: [],
}

const actorGroups = [
  ["Alice", [{ row: rowA, url: "blob:a" }]],
  ["No actor", [{ row: rowB, url: "blob:b" }]],
] as const

describe("CuratedFrameActorsTab", () => {
  it("renders actor groups and passes selection state to each grid", () => {
    const wrapper = mount(CuratedFrameActorsTab, {
      props: {
        actorGroups,
        batchMode: false,
        selectedIds: ["frame-a"],
        nearDuplicateIds: ["frame-b"],
        selectGroupAriaLabel: "Select actor group",
      },
    })
    const grids = wrapper.findAllComponents({ name: "CuratedFrameGrid" })

    expect(wrapper.text()).toContain("Alice")
    expect(wrapper.text()).toContain("No actor")
    expect(wrapper.find("input[type='checkbox']").exists()).toBe(false)
    expect(grids).toHaveLength(2)
    expect(grids[0]!.props("items")).toEqual(actorGroups[0]![1])
    expect(grids[0]!.props("sectionActor")).toBe("Alice")
    expect(grids[0]!.props("selectedIds")).toEqual(["frame-a"])
    expect(grids[0]!.props("nearDuplicateIds")).toEqual(["frame-b"])
  })

  it("forwards group checkbox and frame card events with actor context", async () => {
    const wrapper = mount(CuratedFrameActorsTab, {
      props: {
        actorGroups,
        batchMode: true,
        selectedIds: ["frame-a"],
        nearDuplicateIds: [],
        selectGroupAriaLabel: "Select actor group",
      },
    })
    const checkboxes = wrapper.findAll("input[type='checkbox']")
    const checkboxElements = checkboxes.map((checkbox) => checkbox.element as HTMLInputElement)

    expect(checkboxElements[0]!.checked).toBe(true)
    expect(checkboxes[0]!.attributes("aria-label")).toBe("Select actor group")
    expect(checkboxElements[1]!.checked).toBe(false)

    await checkboxes[1]!.setValue(true)
    await wrapper.findAll("[data-toggle]")[0]!.trigger("click")
    await wrapper.findAll("[data-open]")[0]!.trigger("click")
    await wrapper.findAll("[data-menu]")[0]!.trigger("click")

    expect(wrapper.emitted("groupSelectionChange")?.[0]).toEqual([
      "No actor",
      actorGroups[1]![1],
      true,
    ])
    expect(wrapper.emitted("toggleSelection")).toEqual([["frame-a", "Alice"]])
    expect(wrapper.emitted("open")?.[0]?.[0]).toEqual(actorGroups[0]![1][0])
    expect(wrapper.emitted("open")?.[0]?.[1]).toBe("Alice")
    expect(wrapper.emitted("contextmenu")?.[0]?.[1]).toEqual(actorGroups[0]![1][0])
    expect(wrapper.emitted("contextmenu")?.[0]?.[2]).toBe("Alice")
  })
})
