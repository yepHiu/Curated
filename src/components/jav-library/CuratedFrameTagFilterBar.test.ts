import { defineComponent, h } from "vue"
import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import CuratedFrameTagFilterBar from "./CuratedFrameTagFilterBar.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

const SlotStub = defineComponent({
  name: "SlotStub",
  setup(_, { slots }) {
    return () => h("div", slots.default?.())
  },
})

const ButtonStub = defineComponent({
  name: "ButtonStub",
  inheritAttrs: false,
  setup(_, { attrs, slots, emit }) {
    return () =>
      h(
        "button",
        {
          type: "button",
          ...attrs,
          onClick: (event: MouseEvent) => {
            emit("click", event)
            emit("select", event)
          },
        },
        slots.default?.(),
      )
  },
})

const CheckboxStub = defineComponent({
  name: "CheckboxStub",
  inheritAttrs: false,
  props: {
    checked: { type: Boolean, default: false },
  },
  setup(props, { attrs, slots, emit }) {
    return () =>
      h(
        "button",
        {
          type: "button",
          ...attrs,
          "aria-checked": props.checked ? "true" : "false",
          onClick: (event: MouseEvent) => {
            emit("click", event)
            emit("select", event)
          },
        },
        slots.default?.(),
      )
  },
})

function mountBar(props: {
  facets: { name: string; count: number }[]
  selectedTags: string[]
}) {
  return mount(CuratedFrameTagFilterBar, {
    props,
    global: {
      stubs: {
        DropdownMenu: SlotStub,
        DropdownMenuContent: SlotStub,
        DropdownMenuGroup: SlotStub,
        DropdownMenuItem: ButtonStub,
        DropdownMenuCheckboxItem: CheckboxStub,
        DropdownMenuLabel: SlotStub,
        DropdownMenuSeparator: true,
        DropdownMenuTrigger: SlotStub,
      },
    },
  })
}

describe("CuratedFrameTagFilterBar", () => {
  it("renders a compact dropdown toggle with multi-select actions", async () => {
    const wrapper = mountBar({
      facets: [
        { name: "favorite", count: 4 },
        { name: "close-up", count: 2 },
      ],
      selectedTags: ["close-up"],
    })

    const toggle = wrapper.get("[data-curated-tag-filter-toggle]")
    expect(toggle.text()).toContain("close-up")
    expect(toggle.attributes("aria-pressed")).toBe("true")
    expect(wrapper.get("[data-curated-tag-filter-selected]").text()).toContain("close-up")
    expect(wrapper.get('[data-curated-tag-filter-option="close-up"]').attributes("aria-checked")).toBe(
      "true",
    )
    expect(wrapper.get('[data-curated-tag-filter-option="favorite"]').attributes("aria-checked")).toBe(
      "false",
    )

    await wrapper.get("[data-curated-tag-filter-clear]").trigger("click")
    await wrapper.get('[data-curated-tag-filter-option="favorite"]').trigger("click")

    expect(wrapper.emitted("clear")).toEqual([[]])
    expect(wrapper.emitted("toggleTag")).toEqual([["favorite"]])
  })

  it("keeps the button when facets are empty and shows the empty hint in the menu", () => {
    const wrapper = mountBar({
      facets: [],
      selectedTags: [],
    })

    expect(wrapper.get("[data-curated-tag-filter-toggle]").exists()).toBe(true)
    expect(wrapper.text()).toContain("curated.tagFilterEmpty")
    expect(wrapper.find("[data-curated-tag-filter-option]").exists()).toBe(false)
  })

  it("shows selected count when more than one tag is active", () => {
    const wrapper = mountBar({
      facets: [
        { name: "favorite", count: 4 },
        { name: "close-up", count: 2 },
      ],
      selectedTags: ["favorite", "close-up"],
    })

    expect(wrapper.get("[data-curated-tag-filter-toggle]").text()).toContain(
      'curated.tagFilterSelectedCount:{"count":2}',
    )
    expect(wrapper.get("[data-curated-tag-filter-selected]").text()).toContain("favorite")
    expect(wrapper.get("[data-curated-tag-filter-selected]").text()).toContain("close-up")
  })
})
