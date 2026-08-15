import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import LibrarySavedViewsControls from "./LibrarySavedViewsControls.vue"

const routeMock = vi.hoisted(() => ({
  name: "library",
  path: "/library",
  query: {
    q: "Mina",
    playState: "unwatched",
    resolution: "2160p",
    selected: "movie-1",
    autoplay: "1",
    t: "42",
  } as Record<string, string>,
}))

const routerMock = vi.hoisted(() => ({
  push: vi.fn().mockResolvedValue(undefined),
  replace: vi.fn().mockResolvedValue(undefined),
}))

const serviceMock = vi.hoisted(() => ({
  movies: { value: [] as { year: number; tags: string[]; userTags: string[]; actors: string[]; studio: string }[] },
  savedViews: {
    value: [
      {
        id: "view-1",
        name: "Five stars",
        filters: {
          schemaVersion: 1 as const,
          mode: "favorites" as const,
          tab: "top-rated" as const,
          userRating: 5,
        },
        sortOrder: 0,
        createdAt: "2026-07-20T00:00:00Z",
        updatedAt: "2026-07-20T00:00:00Z",
      },
    ],
  },
  refreshSavedViews: vi.fn().mockResolvedValue(undefined),
  createSavedView: vi.fn(),
  updateSavedView: vi.fn(),
  deleteSavedView: vi.fn(),
  reorderSavedViews: vi.fn(),
}))

vi.mock("vue-router", () => ({
  useRoute: () => routeMock,
  useRouter: () => routerMock,
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => serviceMock,
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: vi.fn(),
}))

const SlotStub = { template: "<div><slot /></div>" }
const ButtonStub = {
  inheritAttrs: false,
  props: ["disabled"],
  emits: ["click"],
  template:
    '<button v-bind="$attrs" :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>',
}
const InputStub = {
  inheritAttrs: false,
  props: ["modelValue"],
  emits: ["update:modelValue"],
  template:
    '<input v-bind="$attrs" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
}

function mountControls(slots?: { default?: string }) {
  return mount(LibrarySavedViewsControls, {
    ...(slots ? { slots } : {}),
    global: {
      stubs: {
        Badge: SlotStub,
        Button: ButtonStub,
        Dialog: SlotStub,
        DialogClose: SlotStub,
        DialogContent: SlotStub,
        DialogDescription: SlotStub,
        DialogFooter: SlotStub,
        DialogHeader: SlotStub,
        DialogTitle: SlotStub,
        DropdownMenu: SlotStub,
        DropdownMenuContent: SlotStub,
        DropdownMenuGroup: SlotStub,
        DropdownMenuItem: ButtonStub,
        DropdownMenuLabel: SlotStub,
        DropdownMenuSeparator: SlotStub,
        DropdownMenuSub: SlotStub,
        DropdownMenuSubContent: SlotStub,
        DropdownMenuSubTrigger: SlotStub,
        DropdownMenuTrigger: SlotStub,
        Input: InputStub,
        Popover: SlotStub,
        PopoverContent: SlotStub,
        PopoverTrigger: SlotStub,
        Select: SlotStub,
        SelectContent: SlotStub,
        SelectGroup: SlotStub,
        SelectItem: SlotStub,
        SelectTrigger: SlotStub,
        SelectValue: SlotStub,
        Bookmark: true,
        Check: true,
        ChevronDown: true,
        ChevronUp: true,
        Filter: true,
        LoaderCircle: true,
        Pencil: true,
        RefreshCw: true,
        Save: true,
        Trash2: true,
        X: true,
      },
    },
  })
}

function buttonByText(wrapper: ReturnType<typeof mountControls>, text: string) {
  const button = wrapper.findAll("button").find((candidate) => candidate.text().trim() === text)
  if (!button) {
    throw new Error(`button not found: ${text}`)
  }
  return button
}

beforeEach(() => {
  serviceMock.refreshSavedViews.mockClear()
  serviceMock.createSavedView.mockReset()
  serviceMock.updateSavedView.mockReset()
  serviceMock.deleteSavedView.mockReset()
  serviceMock.reorderSavedViews.mockReset()
  routerMock.push.mockClear()
  routerMock.replace.mockClear()
})

describe("LibrarySavedViewsControls", () => {
  it("loads views and saves only canonical persistent filters", async () => {
    serviceMock.createSavedView.mockResolvedValue({
      id: "view-created",
      name: "Mina 4K",
      filters: { schemaVersion: 1 },
      sortOrder: 1,
      createdAt: "2026-07-20T00:00:00Z",
      updatedAt: "2026-07-20T00:00:00Z",
    })
    const wrapper = mountControls()
    await flushPromises()
    expect(serviceMock.refreshSavedViews).toHaveBeenCalledTimes(1)

    await buttonByText(wrapper, "library.savedViewSaveCurrent").trigger("click")
    await wrapper.get('input[placeholder="library.savedViewNamePlaceholder"]').setValue("Mina 4K")
    await buttonByText(wrapper, "library.savedViewSave").trigger("click")
    await flushPromises()

    expect(serviceMock.createSavedView).toHaveBeenCalledWith(
      "Mina 4K",
      expect.objectContaining({
        schemaVersion: 1,
        mode: "library",
        q: "Mina",
        playState: "unwatched",
        resolution: "4k",
      }),
    )
    const filters = serviceMock.createSavedView.mock.calls[0]?.[1]
    expect(filters).not.toHaveProperty("selected")
    expect(filters).not.toHaveProperty("autoplay")
    expect(filters).not.toHaveProperty("t")
  })

  it("applies a Saved View by rebuilding a transient-free route target", async () => {
    const wrapper = mountControls()
    await buttonByText(wrapper, "library.savedViewApply").trigger("click")
    await flushPromises()

    expect(routerMock.push).toHaveBeenCalledWith({
      name: "favorites",
      query: {
        tab: "top-rated",
        userRating: "5",
      },
    })
  })

  it("shows dismissible chips for active URL filters", () => {
    const wrapper = mountControls()
    expect(wrapper.find("[data-library-filter-chips]").exists()).toBe(true)
    expect(wrapper.find("[data-library-filter-chip=playState]").exists()).toBe(true)
    expect(wrapper.find("[data-library-filter-chip=resolution]").exists()).toBe(true)
  })

  it("keeps filter, saved views, and trailing actions on one unwrapped row", () => {
    const wrapper = mountControls({
      default: '<button data-trailing-action type="button">batch</button>',
    })
    const row = wrapper.get("[data-library-saved-view-actions]")
    expect(row.classes()).toEqual(
      expect.arrayContaining(["flex", "flex-nowrap", "items-center"]),
    )
    expect(row.find("[data-trailing-action]").exists()).toBe(true)
    expect(row.text()).toContain("library.savedViewFilters")
    expect(row.text()).toContain("library.savedViews")
    expect(wrapper.get("[data-library-filter-chips]").find("[data-trailing-action]").exists()).toBe(false)
  })
})
