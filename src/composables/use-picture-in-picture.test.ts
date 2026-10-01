import { flushPromises, mount } from "@vue/test-utils"
import { defineComponent, h, ref } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { classifyPictureInPictureFailure, usePictureInPicture } from "./use-picture-in-picture"

/** 使用真实 video 元素承载模拟原生 API，验证异步生命周期而非浏览器实现。 */
function setup(ready = true) {
  vi.stubGlobal("document", document)
  const request = vi.fn<() => Promise<PictureInPictureWindow>>()
  const exit = vi.fn().mockImplementation(async () => {
    // 模拟原生退出更新所属元素并发出标准事件。
    const video = document.pictureInPictureElement
    Object.defineProperty(document, "pictureInPictureElement", { configurable: true, value: null })
    video?.dispatchEvent(new Event("leavepictureinpicture"))
  })
  vi.spyOn(document, "pictureInPictureEnabled", "get").mockReturnValue(true)
  Object.defineProperty(document, "pictureInPictureElement", { configurable: true, value: null })
  Object.defineProperty(document, "exitPictureInPicture", { configurable: true, value: exit })
  Object.defineProperty(HTMLVideoElement.prototype, "requestPictureInPicture", { configurable: true, value: request })
  const video = document.createElement("video")
  Object.defineProperty(video, "readyState", { configurable: true, value: ready ? 1 : 0 })
  Object.defineProperty(video, "videoWidth", { configurable: true, value: ready ? 1920 : 0 })
  const source = ref<string | null>("/video.mp4")
  const failure = vi.fn()
  let controls!: ReturnType<typeof usePictureInPicture>
  const wrapper = mount(defineComponent({
    /** 在组件作用域内建立媒体控制器，以验证卸载处理。 */
    setup() {
      controls = usePictureInPicture(ref(video), source, failure)
      return () => { /* 探针不需要 UI，仅保留生命周期。 */ return h("div") }
    },
  }))
  return { wrapper, video, source, request, exit, failure, controls }
}

const originalProperties = ([

  [document, "pictureInPictureEnabled"], [document, "pictureInPictureElement"],
  [document, "exitPictureInPicture"], [HTMLVideoElement.prototype, "requestPictureInPicture"],
] as [object, string][]).map(([object, key]) => {
  // 保存 jsdom 原始接口；测试结束恢复所有属性。
  return { object: object as object, key: key as string, descriptor: Object.getOwnPropertyDescriptor(object, key) }
})
beforeEach(() => {
  // jsdom 没有 PiP，创建可配置的只读能力 getter 以供 mock。
  Object.defineProperty(document, "pictureInPictureEnabled", { configurable: true, get: () => { /* 模拟默认不可用。 */ return false } })
})
afterEach(() => {
  // 恢复浏览器对象，避免影响其它组件测试。
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  for (const original of originalProperties) {
    if (original.descriptor) Object.defineProperty(original.object, original.key, original.descriptor)
    else Reflect.deleteProperty(original.object, original.key)
  }
})

describe("picture-in-picture lifecycle", () => {
  // 进入需要视频元数据；系统已经打开小窗时即使媒体变空仍可退出。
  it("gates entry on metadata and video tracks while keeping exit available", async () => {
    const test = setup(false)
    expect(test.controls.canToggle.value).toBe(false)
    await test.controls.toggle()
    expect(test.request).not.toHaveBeenCalled()
    expect(test.failure).toHaveBeenCalledWith("notReady")
    Object.defineProperty(test.video, "readyState", { configurable: true, value: 1 })
    Object.defineProperty(test.video, "videoWidth", { configurable: true, value: 1920 })
    test.video.dispatchEvent(new Event("loadedmetadata"))
    expect(test.controls.canToggle.value).toBe(true)
    Object.defineProperty(document, "pictureInPictureElement", { configurable: true, value: test.video })
    test.video.dispatchEvent(new Event("enterpictureinpicture"))
    Object.defineProperty(test.video, "readyState", { configurable: true, value: 0 })
    test.video.dispatchEvent(new Event("emptied"))
    expect(test.controls.canToggle.value).toBe(true)
    await test.controls.toggle()
    expect(test.exit).toHaveBeenCalledTimes(1)
    expect(test.controls.active.value).toBe(false)
    test.wrapper.unmount()
  })

  // 双击、快捷键等调用共用同一未决请求锁。
  it("serializes pending requests and reports native rejection", async () => {
    const test = setup()
    let reject!: (error: Error) => void
    test.request.mockImplementation(() => {
      // 保持原生请求未决，验证重入不会产生第二个请求。
      return new Promise((_resolve, rejectRequest) => { /* 保存失败入口。 */ reject = rejectRequest })
    })
    const pending = test.controls.toggle()
    await test.controls.toggle()
    expect(test.request).toHaveBeenCalledTimes(1)
    expect(test.controls.pending.value).toBe(true)
    expect(test.controls.canToggle.value).toBe(false)
    reject(new DOMException("denied", "NotAllowedError"))
    await pending
    expect(test.failure).toHaveBeenCalledWith("gesture")
    expect(test.controls.pending.value).toBe(false)
    test.wrapper.unmount()
  })

  // 原生系统菜单关闭小窗无需点击主页面按钮，状态仍必须同步。
  it("tracks system close and rejects a disabled video", () => {
    const test = setup()
    Object.defineProperty(document, "pictureInPictureElement", { configurable: true, value: test.video })
    test.video.dispatchEvent(new Event("enterpictureinpicture"))
    expect(test.controls.active.value).toBe(true)
    Object.defineProperty(document, "pictureInPictureElement", { configurable: true, value: null })
    test.video.dispatchEvent(new Event("leavepictureinpicture"))
    test.video.disablePictureInPicture = true
    test.video.dispatchEvent(new Event("loadedmetadata"))
    expect(test.controls.active.value).toBe(false)
    expect(test.controls.canEnter.value).toBe(false)
    test.wrapper.unmount()
  })

  // 请求在卸载后才打开小窗时，必须退出旧视频且不发布失败反馈。
  it("cleans a late entry after unmount without touching other videos", async () => {
    const test = setup()
    let resolve!: () => void
    test.request.mockImplementation(() => {
      // 延迟完成以覆盖卸载竞争。
      return new Promise(done => { /* 保存成功入口。 */ resolve = () => { /* 模拟原生成功所属视频。 */ Object.defineProperty(document, "pictureInPictureElement", { configurable: true, value: test.video }); done({} as PictureInPictureWindow) } })
    })
    const pending = test.controls.toggle()
    test.wrapper.unmount()
    resolve()
    await pending
    await flushPromises()
    expect(test.exit).toHaveBeenCalledTimes(1)
    expect(test.failure).not.toHaveBeenCalled()
  })

  // 换源后迟到的完成也必须清理，而不能将新媒体标记成已进入。
  it("invalidates pending entry when the source changes", async () => {
    const test = setup()
    let finish!: (value: PictureInPictureWindow) => void
    test.request.mockImplementation(() => { /* 保存未决原生请求。 */ return new Promise(resolve => { /* 保存完成回调。 */ finish = resolve }) })
    const pending = test.controls.toggle()
    test.source.value = "/other.mp4"
    await flushPromises()
    Object.defineProperty(document, "pictureInPictureElement", { configurable: true, value: test.video })
    finish({} as PictureInPictureWindow)
    await pending
    expect(test.exit).toHaveBeenCalledTimes(1)
    test.wrapper.unmount()
  })

  // 分类不依赖底层浏览器错误文字，适合三种语言共用。
  it.each([
    ["InvalidStateError", "notReady"], ["NotAllowedError", "gesture"],
    ["SecurityError", "blocked"], ["NotSupportedError", "unsupported"], ["Error", "failed"],
  ])("classifies %s as %s", (name, failure) => {
    // 对照浏览器 DOMException 名称验证稳定反馈类型。
    expect(classifyPictureInPictureFailure(new DOMException("test", name))).toBe(failure)
  })
})
