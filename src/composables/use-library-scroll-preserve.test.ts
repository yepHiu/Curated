import { flushPromises, mount } from "@vue/test-utils"
import { defineComponent, ref } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { clearLibraryScrollSnapshot, useLibraryScrollPreserve } from "./use-library-scroll-preserve"

describe("library back-to-top scrolling", () => {
  let frames: Map<number, FrameRequestCallback>
  let nextFrame: number
  let reducedMotion: boolean
  const wrappers: ReturnType<typeof mount>[] = []

  function frame(time: number) {
    const callbacks = [...frames.values()]
    frames.clear()
    callbacks.forEach(callback => callback(time))
  }

  function createScroller() {
    const el = document.createElement("div")
    el.scrollTo = vi.fn((options?: ScrollToOptions | number, y?: number) => {
      el.scrollTop = (typeof options === "number" ? y : options?.top) ?? el.scrollTop
      el.dispatchEvent(new Event("scroll"))
    })
    let scrollToTop = () => {}
    const wrapper = mount(defineComponent({
      setup() {
        const controls = useLibraryScrollPreserve({
          scrollElRef: ref(el),
          preserveKey: ref("back-to-top-test"),
        })
        scrollToTop = controls.scrollToTop
        return () => null
      },
    }))
    wrappers.push(wrapper)
    return { el, scrollToTop, wrapper }
  }

  beforeEach(() => {
    clearLibraryScrollSnapshot("back-to-top-test")
    frames = new Map()
    nextFrame = 0
    reducedMotion = false
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] })
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
      frames.set(++nextFrame, callback)
      return nextFrame
    })
    vi.stubGlobal("cancelAnimationFrame", (id: number) => frames.delete(id))
    vi.stubGlobal("matchMedia", () => ({ matches: reducedMotion }))
  })

  afterEach(() => {
    wrappers.splice(0).forEach(wrapper => wrapper.unmount())
    clearLibraryScrollSnapshot("back-to-top-test")
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it("reaches the top despite virtual-list scroll corrections during the animation", () => {
    const { el, scrollToTop } = createScroller()
    el.scrollTop = 30000
    scrollToTop()
    frame(0)
    frame(150)
    expect(el.scrollTop).toBeLessThan(30000)
    el.scrollTop += 900 // Newly measured chunks adjust the scroll anchor.
    frame(300)
    frame(500)
    expect(el.scrollTop).toBe(0)
    expect(frames.size).toBe(0)
  })

  it.each(["wheel", "touchstart", "pointerdown", "keydown"])("lets %s interrupt back-to-top scrolling", (type) => {
    const { el, scrollToTop } = createScroller()
    el.scrollTop = 30000
    scrollToTop()
    frame(0)
    frame(150)
    el.dispatchEvent(new Event(type))
    el.scrollTop = 15000
    frame(500)
    expect(el.scrollTop).toBe(15000)
    expect(frames.size).toBe(0)
  })

  it("returns immediately when reduced motion is requested", () => {
    reducedMotion = true
    const { el, scrollToTop } = createScroller()
    el.scrollTop = 30000
    scrollToTop()
    expect(el.scrollTop).toBe(0)
    expect(frames.size).toBe(0)
  })

  it("does not reapply an old restored position after clicking back to top", async () => {
    const previous = createScroller()
    previous.el.scrollTop = 30000
    previous.el.dispatchEvent(new Event("scroll"))
    previous.wrapper.unmount()
    const current = createScroller()
    await flushPromises()
    frame(0)
    frame(16)
    await flushPromises()
    expect(current.el.scrollTop).toBe(30000)

    reducedMotion = true
    current.scrollToTop()
    frame(32)
    await vi.advanceTimersByTimeAsync(500)
    expect(current.el.scrollTop).toBe(0)
  })
})
