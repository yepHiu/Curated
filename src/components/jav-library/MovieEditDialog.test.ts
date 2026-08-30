import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { Movie } from "@/domain/movie/types"
import MovieEditDialog from "./MovieEditDialog.vue"

const aiMocks = vi.hoisted(() => ({
  runAction: vi.fn(),
  confirmTool: vi.fn(),
}))

const libraryMocks = vi.hoisted(() => ({
  loadMovieDetail: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: { value: "zh-CN" },
    t: (key: string) => key,
  }),
}))

vi.mock("@/services/ai-service", () => ({
  useAIService: () => aiMocks,
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => libraryMocks,
}))

vi.mock("@/components/ui/button", () => ({
  Button: { name: "Button", template: "<button type='button'><slot /></button>" },
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: { name: "Dialog", template: "<div><slot /></div>", props: ["open"] },
  DialogClose: { name: "DialogClose", template: "<div><slot /></div>" },
  DialogContent: { name: "DialogContent", template: "<div><slot /></div>" },
  DialogDescription: { name: "DialogDescription", template: "<p><slot /></p>" },
  DialogFooter: { name: "DialogFooter", template: "<div><slot /></div>" },
  DialogHeader: { name: "DialogHeader", template: "<header><slot /></header>" },
  DialogTitle: { name: "DialogTitle", template: "<h3><slot /></h3>" },
}))

vi.mock("@/components/ui/input", () => ({
  Input: { name: "Input", template: "<input />" },
}))

const movie: Movie = {
  id: "m1",
  title: "Sample Title",
  code: "ABC-123",
  studio: "Studio",
  actors: [],
  tags: [],
  userTags: [],
  runtimeMinutes: 90,
  rating: 4,
  summary: "Visit ads.example for plot.",
  isFavorite: false,
  addedAt: "2026-01-01",
  location: "C:\\video.mp4",
  resolution: "1080p",
  year: 2024,
  tone: "",
  coverClass: "",
}

function mountDialog(open = true) {
  return mount(MovieEditDialog, {
    props: {
      movie,
      open,
      patchMovieDisplay: vi.fn(),
    },
  })
}

describe("MovieEditDialog AI actions", () => {
  beforeEach(() => {
    aiMocks.runAction.mockReset()
    aiMocks.confirmTool.mockReset()
    libraryMocks.loadMovieDetail.mockReset()
  })
  it("hides in-field AI buttons while the experimental gate is off", () => {
    const wrapper = mountDialog()
    expect(wrapper.find("[data-movie-edit-ai-translate-summary]").exists()).toBe(false)
    expect(wrapper.find("[data-movie-edit-ai-translate]").exists()).toBe(false)
  })

  it("previews a summary translation then applies through confirm", async () => {
    const { useExperimentalAgent } = await import("@/lib/experimental-agent")
    useExperimentalAgent().setEnabled(true)
    aiMocks.runAction.mockResolvedValue({
      action: "translate_summary",
      name: "update_movie_display_overrides",
      sessionId: "act_1",
      originalText: movie.summary,
      proposedText: "Localized plot.",
      confirmToken: "cfm_1",
      arguments: { movieId: "m1", userSummary: "Localized plot." },
    })
    aiMocks.confirmTool.mockResolvedValue({ ok: true, name: "update_movie_display_overrides" })
    libraryMocks.loadMovieDetail.mockResolvedValue(movie)
    const wrapper = mountDialog()
    expect(wrapper.get("[data-movie-edit-summary-field]").find("[data-movie-edit-ai-translate-summary]").exists()).toBe(true)
    await wrapper.get("[data-movie-edit-ai-translate-summary]").trigger("click")
    await flushPromises()
    expect(aiMocks.runAction).toHaveBeenCalledWith("translate_summary", {
      movieId: "m1",
      body: movie.summary,
      locale: "zh-CN",
    })
    expect(aiMocks.runAction).not.toHaveBeenCalledWith("translate_title", expect.anything())
    await wrapper.get("[data-movie-edit-ai-apply]").trigger("click")
    await flushPromises()
    expect(aiMocks.confirmTool).toHaveBeenCalled()
    expect(libraryMocks.loadMovieDetail).toHaveBeenCalledWith("m1")
    useExperimentalAgent().setEnabled(false)
  })

  it("shows a spinner only on the clicked field while a summary action is in flight", async () => {
    const { useExperimentalAgent } = await import("@/lib/experimental-agent")
    useExperimentalAgent().setEnabled(true)
    aiMocks.runAction.mockReturnValue(new Promise(() => {}))
    const wrapper = mountDialog()
    await wrapper.get("[data-movie-edit-ai-translate-summary]").trigger("click")
    await flushPromises()
    expect(wrapper.get("[data-movie-edit-ai-translate-summary]").html()).toContain("animate-spin")
    expect(wrapper.get("[data-movie-edit-ai-translate]").html()).not.toContain("animate-spin")
    expect(aiMocks.runAction).toHaveBeenCalledTimes(1)
    useExperimentalAgent().setEnabled(false)
  })
})
