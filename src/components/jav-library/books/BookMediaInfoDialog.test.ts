import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import BookMediaInfoDialog from "./BookMediaInfoDialog.vue"

const aiMocks = vi.hoisted(() => ({
  runAction: vi.fn(),
  confirmTool: vi.fn(),
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

vi.mock("@/components/ui/button", () => ({
  Button: { name: "Button", template: "<button type='button'><slot /></button>" },
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: { name: "Dialog", template: "<div><slot /></div>", props: ["open"] },
  DialogClose: { name: "DialogClose", template: "<div><slot /></div>" },
  DialogContent: { name: "DialogContent", template: "<section><slot /></section>" },
  DialogDescription: { name: "DialogDescription", template: "<p><slot /></p>" },
  DialogFooter: { name: "DialogFooter", template: "<footer><slot /></footer>" },
  DialogHeader: { name: "DialogHeader", template: "<header><slot /></header>" },
  DialogTitle: { name: "DialogTitle", template: "<h3><slot /></h3>" },
}))

vi.mock("@/components/ui/input", () => ({
  Input: {
    name: "Input",
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template: "<input :value='modelValue' @input=\"$emit('update:modelValue', ($event.target).value)\" />",
  },
}))

const book = {
  title: "A book",
  sourceFileName: "archive.cbz",
  location: "D:/Books/archive.cbz",
  pageCount: 24,
  addedAt: "2026-09-01T00:00:00Z",
  updatedAt: "2026-09-02T00:00:00Z",
}

function mountDialog(open = true) {
  // 挂载媒体信息弹窗，默认漫画册与打开状态。
  return mount(BookMediaInfoDialog, {
    props: {
      open,
      kind: "comics",
      entityId: "comic-1",
      book,
    },
  })
}

describe("BookMediaInfoDialog", () => {
  beforeEach(async () => {
    aiMocks.runAction.mockReset()
    aiMocks.confirmTool.mockReset()
    const { useExperimentalAgent } = await import("@/lib/experimental-agent")
    useExperimentalAgent().setEnabled(false)
  })

  it("shows source details and saves an edited title", async () => {
    const wrapper = mountDialog()
    await flushPromises()
    const dialog = wrapper.get("[data-book-media-info]")
    expect(dialog.findAll("dd").map((row) => row.text())).toEqual([
      "archive.cbz",
      "D:/Books/archive.cbz",
      "CBZ",
      "24",
      new Date("2026-09-01T00:00:00Z").toLocaleString(),
      new Date("2026-09-02T00:00:00Z").toLocaleString(),
    ])
    expect(dialog.find("[data-book-media-title]").exists()).toBe(true)
    await dialog.get("[data-book-media-title]").setValue("展示标题")
    await dialog.get("[data-book-media-save]").trigger("click")
    const save = wrapper.emitted("saveTitle")?.[0] as [string, (err?: unknown) => void]
    expect(save[0]).toBe("展示标题")
    save[1]()
    await flushPromises()
    expect(wrapper.emitted("update:open")).toEqual([[false]])
    wrapper.unmount()
  })

  it("shows placeholders for absent source fields and invalid dates", async () => {
    const wrapper = mount(BookMediaInfoDialog, {
      props: {
        open: true,
        kind: "photos",
        entityId: "photo-1",
        book: { title: "No source", sourceFileName: "", location: "", pageCount: 0, addedAt: "", updatedAt: "invalid" },
      },
    })
    await flushPromises()
    expect(wrapper.findAll("dd").map((row) => row.text())).toEqual(["—", "—", "—", "0", "—", "—"])
    wrapper.unmount()
  })

  it("hides the translate button while Agent writes are off", () => {
    const wrapper = mountDialog()
    expect(wrapper.find("[data-book-media-ai-translate]").exists()).toBe(false)
    wrapper.unmount()
  })

  it("previews and applies a comic title translation", async () => {
    const { useExperimentalAgent } = await import("@/lib/experimental-agent")
    useExperimentalAgent().setEnabled(true)
    aiMocks.runAction.mockResolvedValue({
      action: "translate_title",
      name: "update_comic_title",
      sessionId: "act_1",
      originalText: "A book",
      proposedText: "Localized book",
      confirmToken: "cfm_1",
      arguments: { comicId: "comic-1", title: "Localized book" },
    })
    aiMocks.confirmTool.mockResolvedValue({ ok: true, name: "update_comic_title" })
    const wrapper = mountDialog()
    await wrapper.get("[data-book-media-ai-translate]").trigger("click")
    await flushPromises()
    expect(aiMocks.runAction).toHaveBeenCalledWith("translate_title", {
      comicId: "comic-1",
      body: "A book",
      locale: "zh-CN",
    }, expect.any(AbortSignal))
    await wrapper.get("[data-book-media-ai-apply]").trigger("click")
    await flushPromises()
    expect(aiMocks.confirmTool).toHaveBeenCalled()
    expect(wrapper.emitted("applied")).toEqual([[]])
    useExperimentalAgent().setEnabled(false)
    wrapper.unmount()
  })
})
