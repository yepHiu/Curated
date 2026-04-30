import { flushPromises, mount } from "@vue/test-utils"
import { computed, ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import HistoryView from "./HistoryView.vue"

const playbackRows = vi.hoisted(() => [
  {
    movieId: "movie-1",
    positionSec: 120,
    updatedAt: "2026-04-11T10:00:00.000Z",
  },
])
const removeProgressMock = vi.hoisted(() => vi.fn())
const pushAppToastMock = vi.hoisted(() => vi.fn())

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("zh-CN"),
    t: (key: string, params?: Record<string, unknown>) =>
      params?.n ? `${key}:${String(params.n)}` : key,
  }),
}))

vi.mock("vue-router", () => ({
  RouterLink: { name: "RouterLink", template: "<a><slot /></a>" },
  useRouter: () => ({
    push: vi.fn(),
  }),
}))

vi.mock("@/components/jav-library/PlaybackHistoryCard.vue", () => ({
  default: {
    name: "PlaybackHistoryCard",
    props: ["movie", "entry", "batchMode", "selected"],
    emits: ["click", "remove", "toggleSelect"],
    template: `
      <article
        data-history-card
        :data-movie-id="movie.id"
        :data-selected="String(selected)"
        @click="$emit('click')"
      >
        <button data-history-remove @click.stop="$emit('remove')" />
        <button data-history-toggle-select @click.stop="$emit('toggleSelect')" />
      </article>
    `,
  },
}))

vi.mock("@/components/ui/button", () => ({
  Button: { name: "Button", template: "<button @click=\"$emit('click', $event)\"><slot /></button>" },
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: { name: "Dialog", template: "<div><slot /></div>" },
  DialogClose: { name: "DialogClose", template: "<div><slot /></div>" },
  DialogContent: { name: "DialogContent", template: "<div><slot /></div>" },
  DialogDescription: { name: "DialogDescription", template: "<div><slot /></div>" },
  DialogFooter: { name: "DialogFooter", template: "<div><slot /></div>" },
  DialogHeader: { name: "DialogHeader", template: "<div><slot /></div>" },
  DialogTitle: { name: "DialogTitle", template: "<div><slot /></div>" },
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: pushAppToastMock,
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    getMovieById: vi.fn((id: string) => ({
      id,
      title: `Movie ${id}`,
    })),
    movies: computed(() => []),
  }),
}))

vi.mock("@/lib/playback-history-groups", () => ({
  groupPlaybackRowsByLocalDay: vi.fn((rows: unknown[]) => [
    {
      dayKey: "2026-04-11",
      label: "Today",
      rows,
    },
  ]),
}))

vi.mock("@/lib/player-route", () => ({
  buildPlayerRouteFromHistory: vi.fn(() => ({ name: "player" })),
}))

vi.mock("@/lib/playback-progress-storage", () => ({
  playbackProgressRevision: ref(0),
  listSortedByUpdatedDesc: vi.fn(() => playbackRows),
  removeProgress: removeProgressMock,
}))

describe("HistoryView", () => {
  beforeEach(() => {
    playbackRows.splice(0, playbackRows.length, {
      movieId: "movie-1",
      positionSec: 120,
      updatedAt: "2026-04-11T10:00:00.000Z",
    })
    removeProgressMock.mockReset()
    removeProgressMock.mockResolvedValue(undefined)
    pushAppToastMock.mockReset()
  })

  it("renders the empty state when no playback rows are available", () => {
    playbackRows.splice(0, playbackRows.length)

    const wrapper = mount(HistoryView)

    expect(wrapper.text()).toContain("history.empty")
    expect(wrapper.find("[data-history-card]").exists()).toBe(false)
  })

  it("removes a single playback history row after confirmation", async () => {
    const wrapper = mount(HistoryView)

    await wrapper.get("[data-history-remove]").trigger("click")
    const confirmButton = wrapper
      .findAll("button")
      .find((button) => button.text().includes("history.deleteAction"))

    expect(confirmButton).toBeDefined()
    await confirmButton!.trigger("click")
    await flushPromises()

    expect(removeProgressMock).toHaveBeenCalledWith("movie-1")
    expect(pushAppToastMock).toHaveBeenCalledWith("history.deleteSuccess", {
      variant: "success",
      durationMs: 3200,
    })
  })

  it("removes selected rows in batch mode", async () => {
    const wrapper = mount(HistoryView)

    const enterButton = wrapper
      .findAll("button")
      .find((button) => button.text().includes("history.batchManage"))

    expect(enterButton).toBeDefined()
    await enterButton!.trigger("click")
    await wrapper.get("[data-history-toggle-select]").trigger("click")

    expect(wrapper.get("[data-history-card]").attributes("data-selected")).toBe("true")

    const batchDeleteButton = wrapper
      .findAll("button")
      .find((button) => button.text().includes("history.batchDeleteAction"))

    expect(batchDeleteButton).toBeDefined()
    await batchDeleteButton!.trigger("click")
    await flushPromises()

    expect(removeProgressMock).toHaveBeenCalledWith("movie-1")
    expect(pushAppToastMock).toHaveBeenCalledWith("history.batchDeleteSummary", {
      variant: "success",
    })
  })

  it("aligns flush with the content bottom when batch mode is active", async () => {
    const wrapper = mount(HistoryView)

    const buttons = wrapper.findAll("button")
    const enterButton = buttons.at(0)

    expect(enterButton).toBeTruthy()
    await enterButton!.trigger("click")

    const toolbar = wrapper.get('[role="toolbar"]')
    expect(toolbar.classes().join(" ")).not.toContain("rounded-b-[calc")
  })
})
