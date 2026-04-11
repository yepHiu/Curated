import { mount } from "@vue/test-utils"
import { computed, ref } from "vue"
import { describe, expect, it, vi } from "vitest"
import HistoryView from "./HistoryView.vue"

const playbackRows = vi.hoisted(() => [
  {
    movieId: "movie-1",
    positionSec: 120,
    updatedAt: "2026-04-11T10:00:00.000Z",
  },
])

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
  default: { name: "PlaybackHistoryCard", template: "<article />" },
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
  pushAppToast: vi.fn(),
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
  removeProgress: vi.fn(),
}))

describe("HistoryView batch toolbar layout", () => {
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
