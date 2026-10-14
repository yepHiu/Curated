/** 滑块轨道换算参数。拇指宽度取按下时的尺寸，避免拖动过程中拇指布局变化把数值带偏。 */
export interface SliderPointerScale {
  left: number
  width: number
  thumbWidth: number
  min: number
  max: number
  step: number
}

/** 把指针横坐标换成滑块数值。grabOffset 是相对轨道左缘要扣掉的像素，轨道点击用拇指半宽，按住拇指时用按下点。 */
export function scaleSliderPointer(clientX: number, grabOffset: number, scale: SliderPointerScale): number {
  const span = Math.max(1, scale.width - Math.max(0, scale.thumbWidth))
  const ratio = Math.min(1, Math.max(0, (clientX - scale.left - grabOffset) / span))
  const raw = scale.min + ratio * (scale.max - scale.min)
  const step = scale.step
  if (!(step > 0) || !Number.isFinite(raw)) return Math.min(scale.max, Math.max(scale.min, raw))
  const snapped = scale.min + Math.round((raw - scale.min) / step) * step
  const clamped = Math.min(scale.max, Math.max(scale.min, snapped))
  const decimals = step.toString().includes("e") ? 0 : (step.toString().split(".")[1]?.length ?? 0)
  const factor = 10 ** Math.min(decimals, 8)
  return Math.round(clamped * factor) / factor
}

/** 一次滑块拖动：只跟踪开始时的那根指针，移动和松开沿用同一个抓取偏移。 */
export function createSliderPointerDrag() {
  let pointerId: number | undefined
  let grabOffset = 0
  let thumbWidth = 0
  let min = 0
  let max = 100
  let step = 1

  /** 是否正按住滑块。 */
  function active() {
    return pointerId !== undefined
  }

  /** 返回正在跟踪的指针编号，供松开和失焦核对。 */
  function currentPointerId() {
    return pointerId
  }

  /** 用按下时记下的步长、拇指宽度和抓取偏移，加上当前轨道位置换算数值。 */
  function valueAt(clientX: number, left: number, width: number) {
    return scaleSliderPointer(clientX, grabOffset, { left, width, thumbWidth, min, max, step })
  }

  /** 开始拖动并返回按下位置对应的数值。 */
  function begin(id: number, clientX: number, offset: number, scale: SliderPointerScale) {
    pointerId = id
    grabOffset = offset
    thumbWidth = scale.thumbWidth
    min = scale.min
    max = scale.max
    step = scale.step
    return valueAt(clientX, scale.left, scale.width)
  }

  /** 同一根指针移动时返回新数值；其它指针不改变当前拖动。 */
  function update(id: number, clientX: number, left: number, width: number) {
    if (id !== pointerId) return
    return valueAt(clientX, left, width)
  }

  /** 结束这根指针的拖动。给出坐标时返回松开位置，否则只取消跟踪。 */
  function finish(id: number, clientX?: number, left?: number, width?: number) {
    if (id !== pointerId) return
    const value = clientX === undefined || left === undefined || width === undefined
      ? undefined
      : valueAt(clientX, left, width)
    pointerId = undefined
    return value
  }

  return { active, currentPointerId, begin, update, finish }
}
