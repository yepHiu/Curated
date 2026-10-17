import { defineComponent, reactive } from "vue"
import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { AIActionPreviewDTO, AIToolApplyDTO } from "@/api/types"
import { useBookMetadata, type MetadataBook } from "./use-book-metadata"

function preview(id: string): AIActionPreviewDTO {
  return { action: "recognize_book_metadata", name: "update_book_metadata", sessionId: id, confirmToken: `token-${id}`, arguments: { metadata: { cleanTitle: `原文 ${id}`, translatedTitle: `译文 ${id}` } } }
}
function harness() {
  const service = { runAction: vi.fn(async (_name: string, body: { comicId?: string }) => preview(body.comicId ?? "1")), confirmTool: vi.fn(async (): Promise<AIToolApplyDTO> => ({ ok: true, name: "update_book_metadata" })) }
  const refresh = vi.fn(async () => undefined)
  let state!: ReturnType<typeof useBookMetadata>
  const wrapper = mount(defineComponent({ setup() { state = useBookMetadata(service, refresh); return () => null } }))
  const books = reactive(Array.from({ length: 7 }, (_, i): MetadataBook => ({ id: String(i), title: `Book ${i}`, sourceFileName: `${i}.zip`, metadata: { cleanTitle: "existing" } })))
  state.reset(books)
  return { service, state, wrapper, refresh }
}

describe("book metadata review", () => {
  it("generates only the current group, never writes before confirmation, and keeps successes on retry", async () => {
    const { service, state, wrapper, refresh } = harness()
    service.runAction.mockRejectedValueOnce(new Error("provider failed"))
    await state.generate("comic", "zh-CN")
    expect(service.runAction).toHaveBeenCalledTimes(5)
    expect(service.confirmTool).not.toHaveBeenCalled()
    expect(state.rows.value[0]?.status).toBe("failed")
    expect(state.rows.value[5]?.status).toBe("pending")
    await state.generate("comic", "zh-CN")
    expect(service.runAction).toHaveBeenCalledTimes(6)
    state.rows.value[1]!.selected = false
    service.confirmTool.mockRejectedValueOnce(new Error("response lost"))
    await state.apply()
    expect(refresh).toHaveBeenCalledTimes(3)
    expect(state.rows.value[0]?.status).toBe("ready")
    const firstToken = service.confirmTool.mock.calls[0]
    await state.apply()
    expect(service.confirmTool.mock.calls[4]).toEqual(firstToken)
    expect(refresh).toHaveBeenCalledTimes(4)
    expect(state.rows.value[1]?.status).toBe("ready")
    wrapper.unmount()
  })

  it("cancels generation and discards a late provider response", async () => {
    const { service, state, wrapper } = harness()
    let finish!: (value: AIActionPreviewDTO) => void
    service.runAction.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    const work = state.generate("comic", "ja")
    state.cancel()
    finish(preview("0"))
    await work
    expect(state.rows.value[0]?.status).toBe("pending")
    expect(state.rows.value[0]?.preview).toBeUndefined()
    expect(service.runAction).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it("stops queued writes but waits for the accepted confirmation", async () => {
    const { service, state, wrapper } = harness()
    await state.generate("comic", "zh-CN")
    let finish!: (value: AIToolApplyDTO) => void
    service.confirmTool.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    const work = state.apply()
    state.cancel()
    finish({ ok: true, name: "update_book_metadata" })
    await work
    expect(service.confirmTool).toHaveBeenCalledTimes(1)
    expect(state.rows.value[0]?.status).toBe("applied")
    expect(state.rows.value[1]?.status).toBe("ready")
    wrapper.unmount()
  })
})
