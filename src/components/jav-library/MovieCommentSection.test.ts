import { flushPromises, mount } from "@vue/test-utils"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { MAX_MOVIE_COMMENT_RUNES } from "@/api/types"
import MovieCommentSection from "./MovieCommentSection.vue"

const serviceMocks = vi.hoisted(() => ({
  getMovieComment: vi.fn(),
  putMovieComment: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: { value: "en" },
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => serviceMocks,
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    emits: ["click"],
    template: "<button v-bind=\"$attrs\" @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/card", () => ({
  Card: { name: "Card", template: "<section><slot /></section>" },
  CardContent: { name: "CardContent", template: "<div><slot /></div>" },
  CardDescription: { name: "CardDescription", template: "<p><slot /></p>" },
  CardHeader: { name: "CardHeader", template: "<header><slot /></header>" },
  CardTitle: { name: "CardTitle", template: "<h2><slot /></h2>" },
}))

async function mountComment(props: { movieId?: string; readonly?: boolean } = {}) {
  const wrapper = mount(MovieCommentSection, {
    props: {
      movieId: props.movieId ?? "movie-1",
      readonly: props.readonly ?? false,
    },
  })
  await flushPromises()
  return wrapper
}

describe("MovieCommentSection", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    serviceMocks.getMovieComment.mockReset()
    serviceMocks.putMovieComment.mockReset()
    serviceMocks.getMovieComment.mockResolvedValue({
      body: "saved note",
      updatedAt: "2026-05-11T12:00:00Z",
    })
    serviceMocks.putMovieComment.mockImplementation((_movieId: string, body: { body: string }) =>
      Promise.resolve({
        body: body.body,
        updatedAt: "2026-05-11T12:01:00Z",
      }),
    )
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("loads the saved comment without auto-saving it back", async () => {
    await mountComment()

    await vi.advanceTimersByTimeAsync(5000)
    await flushPromises()

    expect(serviceMocks.getMovieComment).toHaveBeenCalledWith("movie-1")
    expect(serviceMocks.putMovieComment).not.toHaveBeenCalled()
  })

  it("auto-saves an edited comment after the debounce interval", async () => {
    const wrapper = await mountComment()

    await wrapper.get("textarea").setValue("saved note plus edit")
    await vi.advanceTimersByTimeAsync(799)
    await flushPromises()

    expect(serviceMocks.putMovieComment).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(1)
    await flushPromises()

    expect(serviceMocks.putMovieComment).toHaveBeenCalledWith("movie-1", {
      body: "saved note plus edit",
    })
  })

  it("flushes the previous movie draft before loading a different movie", async () => {
    serviceMocks.getMovieComment.mockImplementation((movieId: string) =>
      Promise.resolve({
        body: `${movieId} saved`,
        updatedAt: "2026-05-11T12:00:00Z",
      }),
    )
    const wrapper = await mountComment({ movieId: "movie-1" })

    await wrapper.get("textarea").setValue("movie-1 edited")
    await wrapper.setProps({ movieId: "movie-2" })
    await flushPromises()

    expect(serviceMocks.putMovieComment).toHaveBeenCalledWith("movie-1", {
      body: "movie-1 edited",
    })
    expect(serviceMocks.getMovieComment).toHaveBeenLastCalledWith("movie-2")
    expect((wrapper.get("textarea").element as HTMLTextAreaElement).value).toBe("movie-2 saved")
  })

  it("queues the latest draft while a save is in flight without overwriting new input", async () => {
    let resolveFirstSave: ((value: { body: string; updatedAt: string }) => void) | undefined
    serviceMocks.putMovieComment
      .mockImplementationOnce(
        () =>
          new Promise<{ body: string; updatedAt: string }>((resolve) => {
            resolveFirstSave = resolve
          }),
      )
      .mockImplementationOnce((_movieId: string, body: { body: string }) =>
        Promise.resolve({
          body: body.body,
          updatedAt: "2026-05-11T12:02:00Z",
        }),
      )
    const wrapper = await mountComment()

    await wrapper.get("textarea").setValue("first edit")
    await vi.advanceTimersByTimeAsync(800)
    await flushPromises()
    await wrapper.get("textarea").setValue("second edit")
    await vi.advanceTimersByTimeAsync(800)
    await flushPromises()

    expect(serviceMocks.putMovieComment).toHaveBeenCalledTimes(1)

    resolveFirstSave?.({
      body: "first edit",
      updatedAt: "2026-05-11T12:01:00Z",
    })
    await flushPromises()
    await flushPromises()

    expect(serviceMocks.putMovieComment).toHaveBeenNthCalledWith(1, "movie-1", {
      body: "first edit",
    })
    expect(serviceMocks.putMovieComment).toHaveBeenNthCalledWith(2, "movie-1", {
      body: "second edit",
    })
    expect((wrapper.get("textarea").element as HTMLTextAreaElement).value).toBe("second edit")
  })

  it("does not save comments over the rune limit", async () => {
    const wrapper = await mountComment()

    await wrapper.get("textarea").setValue("あ".repeat(MAX_MOVIE_COMMENT_RUNES + 1))
    await vi.advanceTimersByTimeAsync(800)
    await flushPromises()

    expect(serviceMocks.putMovieComment).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain("detailPage.commentTooLong")
  })

  it("does not auto-save readonly comments", async () => {
    const wrapper = await mountComment({ readonly: true })

    await vi.advanceTimersByTimeAsync(5000)
    await flushPromises()

    expect(serviceMocks.putMovieComment).not.toHaveBeenCalled()
    expect(wrapper.find("[data-comment-save]").exists()).toBe(false)
    expect(wrapper.get("textarea").attributes("readonly")).toBeDefined()
  })

  it("saves immediately from the manual save button and shows saved feedback", async () => {
    const wrapper = await mountComment()

    await wrapper.get("textarea").setValue("manual edit")
    expect(wrapper.text()).toContain("detailPage.commentUnsaved")

    await wrapper.get("[data-comment-save]").trigger("click")
    await flushPromises()

    expect(serviceMocks.putMovieComment).toHaveBeenCalledWith("movie-1", {
      body: "manual edit",
    })
    expect(wrapper.text()).toContain("detailPage.commentAutoSaved")
  })
})
