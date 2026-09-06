import { onMounted, onBeforeUnmount } from "vue"
import { applyAIGovernance } from "@/lib/experimental-agent"
import { useAIGovernanceService } from "@/services/ai-governance-service"

/** Refresh global policy when returning from another client; no background polling. */
export function useAIGovernanceSync() {
  let disposed = false
  let revision = 0
  const sync = async () => {
    const current = ++revision
    try {
      const settings = await useAIGovernanceService().getSettings()
      if (!disposed && current === revision) applyAIGovernance(settings)
    } catch { /* Locked/offline clients keep presentation state; backend enforces policy. */ }
  }
  onMounted(() => { window.addEventListener("focus", sync) })
  onBeforeUnmount(() => { disposed = true; window.removeEventListener("focus", sync) })
}
