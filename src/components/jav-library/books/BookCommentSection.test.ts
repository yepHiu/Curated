import { flushPromises, mount } from "@vue/test-utils"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { MAX_BOOK_COMMENT_RUNES } from "@/api/types"
import BookCommentSection from "./BookCommentSection.vue"

const comicMocks = vi.hoisted(() => ({
  getComicComment: vi.fn(),
  putComicComment: vi.fn(),
}))

const photoMocks = vi.hoisted(() => ({
  getPhotoComment: vi.fn(),
  putPhotoComment: vi.fn(),
}))

const aiMocks = vi.hoisted(() => ({
  runAction: vi.fn(),
  confirmTool: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: { value: "en" },
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

vi.mock("@/services/comic-library-service", () => ({
  useComicLibraryService: () => comicMocks,
}))

vi.mock("@/services/photo-library-service", () => ({
  usePhotoLibraryService: () => photoMocks,
}))

vi.mock("@/services/ai-service", () => ({
  useAIService: () => aiMocks,
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: { name: "Dialog", template: "<div v-if='open'><slot /></div>", props: ["open"] },
  DialogContent: { name: "DialogContent", template: "<div><slot /></div>" },
  DialogDescription: { name: "DialogDescription", template: "<p><slot /></p>" },
  DialogFooter: { name: "DialogFooter", template: "<div><slot /></div>" },
  DialogHeader: { name: "DialogHeader", template: "<header><slot /></header>" },
  DialogTitle: { name: "DialogTitle", template: "<h3><slot /></h3>" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: { name: "Button", template: "<button type='button'><slot /></button>" },
}))

vi.mock("@/components/ui/card", () => ({
  Card: { name: "Card", template: "<section><slot /></section>" },
  CardContent: { name: "CardContent", template: "<div><slot /></div>" },
  CardHeader: { name: "CardHeader", template: "<header><slot /></header>" },
  CardTitle: { name: "CardTitle", template: "<h2><slot /></h2>" },
}))

/** 挂载共享备注卡片并等待首次加载。 */
async function mountComment(
  props: { kind?: "comics" | "photos"; entityId?: string } = {},
) {
  const wrapper = mount(BookCommentSection, {
    props: {
      kind: props.kind ?? "comics",
      entityId: props.entityId ?? "comic-1",
    },
  })
  await flushPromises()
  return wrapper
}

describe("BookCommentSection", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    comicMocks.getComicComment.mockReset()
    comicMocks.putComicComment.mockReset()
    photoMocks.getPhotoComment.mockReset()
    photoMocks.putPhotoComment.mockReset()
    comicMocks.getComicComment.mockResolvedValue({
      body: "saved note",
      updatedAt: "2026-09-12T12:00:00Z",
    })
    comicMocks.putComicComment.mockImplementation((_id: string, body: { body: string }) =>
      Promise.resolve({
        body: body.body,
        updatedAt: "2026-09-12T12:01:00Z",
      }),
    )
    photoMocks.getPhotoComment.mockResolvedValue({
      body: "photo note",
      updatedAt: "2026-09-12T12:00:00Z",
    })
    photoMocks.putPhotoComment.mockImplementation((_id: string, body: { body: string }) =>
      Promise.resolve({
        body: body.body,
        updatedAt: "2026-09-12T12:01:00Z",
      }),
    )
    aiMocks.runAction.mockReset()
    aiMocks.confirmTool.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("loads the saved comic comment without auto-saving it back", async () => {
    await mountComment()

    await vi.advanceTimersByTimeAsync(5000)
    await flushPromises()

    expect(comicMocks.getComicComment).toHaveBeenCalledWith("comic-1")
    expect(comicMocks.putComicComment).not.toHaveBeenCalled()
    expect(photoMocks.getPhotoComment).not.toHaveBeenCalled()
  })

  it("auto-saves an edited comic comment after the debounce interval", async () => {
    const wrapper = await mountComment()
    const textarea = wrapper.get("textarea")
    await textarea.setValue("edited note")
    await vi.advanceTimersByTimeAsync(800)
    await flushPromises()

    expect(comicMocks.putComicComment).toHaveBeenCalledWith("comic-1", {
      body: "edited note",
    })
  })

  it("loads and saves photo comments through the photo service", async () => {
    const wrapper = await mountComment({ kind: "photos", entityId: "photo-1" })
    expect(photoMocks.getPhotoComment).toHaveBeenCalledWith("photo-1")
    expect(comicMocks.getComicComment).not.toHaveBeenCalled()

    await wrapper.get("textarea").setValue("new photo note")
    await vi.advanceTimersByTimeAsync(800)
    await flushPromises()

    expect(photoMocks.putPhotoComment).toHaveBeenCalledWith("photo-1", {
      body: "new photo note",
    })
  })

  it("does not save when the draft exceeds the rune limit", async () => {
    const wrapper = await mountComment()
    await wrapper.get("textarea").setValue("あ".repeat(MAX_BOOK_COMMENT_RUNES + 1))
    await vi.advanceTimersByTimeAsync(800)
    await flushPromises()

    expect(comicMocks.putComicComment).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain("detailPage.commentTooLong")
  })

  it("hides AI comment actions while the experimental gate is off", async () => {
    const wrapper = await mountComment()
    expect(wrapper.find("[data-comment-ai-actions]").exists()).toBe(false)
  })

  it("previews a comic polish action through save_comic_comment", async () => {
    const { useExperimentalAgent } = await import("@/lib/experimental-agent")
    useExperimentalAgent().setEnabled(true)
    aiMocks.runAction.mockResolvedValue({
      action: "polish_comment",
      name: "save_comic_comment",
      sessionId: "act_1",
      originalText: "saved note",
      proposedText: "polished note",
      confirmToken: "cfm_1",
      arguments: { comicId: "comic-1", body: "polished note" },
    })
    aiMocks.confirmTool.mockResolvedValue({
      replayed: false,
      ok: true,
      name: "save_comic_comment",
      data: { body: "polished note", updatedAt: "2026-09-12T12:02:00Z" },
    })
    const wrapper = await mountComment()
    expect(wrapper.find("[data-comment-ai-actions]").exists()).toBe(true)
    await wrapper.get("[data-comment-ai-polish]").trigger("click")
    await flushPromises()
    expect(aiMocks.runAction).toHaveBeenCalledWith("polish_comment", {
      comicId: "comic-1",
      body: "saved note",
    }, expect.any(AbortSignal))
    await wrapper.get("[data-comment-ai-apply]").trigger("click")
    await flushPromises()
    expect(aiMocks.confirmTool).toHaveBeenCalled()
    expect((wrapper.get("textarea").element as HTMLTextAreaElement).value).toBe("polished note")
    await vi.advanceTimersByTimeAsync(5000)
    expect(comicMocks.putComicComment).not.toHaveBeenCalled()
    useExperimentalAgent().setEnabled(false)
  })
})
