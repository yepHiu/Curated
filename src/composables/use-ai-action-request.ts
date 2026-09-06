import { onBeforeUnmount, ref } from "vue"
import type { AIActionRequestBody } from "@/api/types"
import { AIServiceError, type AIService } from "@/services/contracts/ai-service"

export function useAIActionRequest(service: AIService) {
  const pending = ref(false)
  let controller: AbortController | undefined
  const cancel = () => controller?.abort()
  onBeforeUnmount(cancel)
  const run = async (name: string, body: AIActionRequestBody) => {
    cancel()
    const request = new AbortController()
    controller = request
    pending.value = true
    try {
      const result = await service.runAction(name, body, request.signal)
      if (request.signal.aborted) throw new AIServiceError("Cancelled", "AI_CANCELLED")
      return result
    } catch (err) {
      if (request.signal.aborted) throw new AIServiceError("Cancelled", "AI_CANCELLED")
      throw err
    } finally {
      if (controller === request) { pending.value = false; controller = undefined }
    }
  }
  return { run, pending, cancel }
}
