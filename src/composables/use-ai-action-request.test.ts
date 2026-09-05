import { defineComponent } from "vue"
import { mount } from "@vue/test-utils"
import { expect, it, vi } from "vitest"
import { useAIActionRequest } from "./use-ai-action-request"
import type { AIService } from "@/services/contracts/ai-service"

it("cancels on unmount and ignores a late provider result", async () => {
  let finish!: (value: unknown) => void
  const runAction = vi.fn(() => new Promise(resolve => { finish = resolve }))
  let action!: ReturnType<typeof useAIActionRequest>
  const wrapper = mount(defineComponent({
    setup() { action = useAIActionRequest({ runAction } as unknown as AIService); return () => null },
  }))
  const pending = expect(action.run("translate_title", { movieId: "m1" })).rejects.toMatchObject({ code: "AI_CANCELLED" })
  wrapper.unmount()
  expect((runAction.mock.calls as unknown as [string, unknown, AbortSignal][])[0]?.[2].aborted).toBe(true)
  finish({ proposedText: "stale text" })
  await pending
  expect(action.pending.value).toBe(false)
})
