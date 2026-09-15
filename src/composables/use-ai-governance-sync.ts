import { onMounted, onBeforeUnmount } from "vue"
import { applyAIGovernance } from "@/lib/experimental-agent"
import { useAIGovernanceService } from "@/services/ai-governance-service"

/** 启动时拉取全局 AI 权限，并在窗口重新获得焦点时同步；不轮询。 */
export function useAIGovernanceSync() {
  let disposed = false
  let revision = 0
  /** 从后端读取治理设置并更新 Agent 入口展示状态。 */
  const sync = async () => {
    const current = ++revision
    try {
      const settings = await useAIGovernanceService().getSettings()
      if (!disposed && current === revision) applyAIGovernance(settings)
    } catch { /* Locked/offline clients keep presentation state; backend enforces policy. */ }
  }
  onMounted(() => {
    void sync()
    window.addEventListener("focus", sync)
  })
  onBeforeUnmount(() => {
    disposed = true
    window.removeEventListener("focus", sync)
  })
}
