import { ref } from "vue"

/** 变更后递增，供列表 computed 依赖刷新 */
export const curatedFramesRevision = ref(0)

export function bumpCuratedFramesRevision() {
  curatedFramesRevision.value += 1
}
