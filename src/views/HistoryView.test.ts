import { flushPromises, mount } from "@vue/test-utils"
import { computed, ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import HistoryView from "./HistoryView.vue"
import type { PlaybackProgressEntry } from "@/lib/playback-progress-storage"

const playbackRows = vi.hoisted((): PlaybackProgressEntry[] => [
  {
    movieId: "movie-1",
    positionSec: 120,
    durationSec: 7200,
    updatedAt: "2026-04-11T10:00:00.000Z",
  },
])
const removeProgressMock = vi.hoisted(() => vi.fn())
const pushAppToastMock = vi.hoisted(() => vi.fn())
const routerPushMock = vi.hoisted(() => vi.fn())
const historyRouteMock = vi.hoisted(() => vi.fn(/* 仅替代路由构造，保留调用参数以验证续播。 */ () => ({ name: "player" })))
const routeQuery = vi.hoisted((): { category?: string } => ({}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("zh-CN"),
    t: (key: string, params?: Record<string, unknown>) =>
      params?.n ? `${key}:${String(params.n)}` : key,
  }),
}))

vi.mock("vue-router", () => ({
  useRoute: () => ({ query: routeQuery }),
  RouterLink: { name: "RouterLink", template: "<a><slot /></a>" },
  useRouter: () => ({
    push: routerPushMock,
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
    getMovieById: vi.fn(/* 混合资料库为历史测试提供两类番号。 */ (id: string) => ({
      id,
      title: `Movie ${id}`,
      code: id === "fc2-movie" ? "FC2-PPV-1234567" : "ABC-123",
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
  buildPlayerRouteFromHistory: historyRouteMock,
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
      durationSec: 7200,
      updatedAt: "2026-04-11T10:00:00.000Z",
    })
    removeProgressMock.mockReset()
    removeProgressMock.mockResolvedValue(undefined)
    pushAppToastMock.mockReset()
    routerPushMock.mockReset()
    historyRouteMock.mockClear()
    delete routeQuery.category
  })

  it("keeps the page title separated from the shell divider", () => {
    const wrapper = mount(HistoryView)
    const content = wrapper.get("[data-history-content]")

    expect(content.classes()).toContain("pt-[var(--app-page-py)]")
    expect(content.classes()).toContain("lg:pt-[var(--app-page-py-lg)]")
  })

  it("renders the empty state when no playback rows are available", () => {
    playbackRows.splice(0, playbackRows.length)

    const wrapper = mount(HistoryView)

    expect(wrapper.text()).toContain("history.empty")
    expect(wrapper.find("[data-history-card]").exists()).toBe(false)
  })

  it.each([undefined, "fc2", "library"])("shows both categories in time order with legacy category=%s", (category) => {
    // 旧类别链接不能隐藏另一类历史，页面也不再提供类别选择器。
    routeQuery.category = category
    playbackRows.unshift({ movieId: "fc2-movie", fileId: "fc2-part-2", positionSec: 240, durationSec: 7200, updatedAt: "2026-04-11T11:00:00.000Z" })
    const wrapper = mount(HistoryView)
    expect(wrapper.findAll("[data-history-card]").map(/* 检查混合列表没有改变时间顺序。 */ (card) => card.attributes("data-movie-id"))).toEqual(["fc2-movie", "movie-1"])
    expect(wrapper.find('[role="combobox"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it.each([
    { movieId: "movie-1", sourceMode: "library" },
    { movieId: "fc2-movie", sourceMode: "fc2" },
  ])("resumes $movieId at its recorded file and time", async ({ movieId, sourceMode }) => {
    // 同一历史页续播两类作品时，保留分部、时间和作品的播放队列来源。
    playbackRows.splice(0, playbackRows.length, { movieId, fileId: "part-2", positionSec: 240.8, durationSec: 7200, updatedAt: "2026-04-11T11:00:00.000Z" })
    const wrapper = mount(HistoryView)
    await wrapper.get("[data-history-card]").trigger("click")
    expect(historyRouteMock).toHaveBeenCalledWith(movieId, 240, "part-2", sourceMode)
    expect(routerPushMock).toHaveBeenCalledWith({ name: "player" })
    wrapper.unmount()
  })

  it("selects and removes ordinary and FC2 history together in batch mode", async () => {
    // 全选范围覆盖统一历史列表内的两类记录。
    playbackRows.push({ movieId: "fc2-movie", fileId: "fc2-part-2", positionSec: 240, durationSec: 7200, updatedAt: "2026-04-11T09:00:00.000Z" })
    const wrapper = mount(HistoryView)
    await wrapper.findAll("button").find(/* 进入既有批量管理。 */ (button) => button.text() === "history.batchManage")!.trigger("click")
    await wrapper.findAll("button").find(/* 全选当前混合历史列表。 */ (button) => button.text() === "history.batchSelectVisible")!.trigger("click")
    expect(wrapper.findAll("[data-history-card]").every(/* 两类记录均已选中。 */ (card) => card.attributes("data-selected") === "true")).toBe(true)
    await wrapper.findAll("button").find(/* 确认删除已选的两类历史记录。 */ (button) => button.text() === "history.batchDeleteAction")!.trigger("click")
    await flushPromises()
    expect(removeProgressMock.mock.calls).toEqual([["movie-1"], ["fc2-movie"]])
    wrapper.unmount()
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
