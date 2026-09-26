import { computed, ref } from "vue"

// Renderer-session only: never persist a simulated connection into server or Desktop configuration.
const requested = ref(false)
export const devRemoteSimulation = computed(() => import.meta.env.DEV && requested.value)

export function setDevRemoteSimulation(enabled: boolean) {
  requested.value = import.meta.env.DEV && enabled
}
