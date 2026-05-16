import { mount } from "@vue/test-utils"
import { nextTick } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const carouselApi = vi.hoisted(() => ({
  scrollPrev: vi.fn(),
  scrollNext: vi.fn(),
  scrollTo: vi.fn(),
  reInit: vi.fn(),
  selectedScrollSnap: vi.fn(() => 0),
  on: vi.fn(),
  off: vi.fn(),
}))

vi.mock("embla-carousel-vue", async () => {
  const { ref } = await vi.importActual<typeof import("vue")>("vue")
  return {
    default: () => [ref(null), ref(carouselApi)],
  }
})

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (key === "preview.title") return `preview.title:${String(params?.code ?? "")}`.trim()
      if (key === "preview.imageOf") return `preview.imageOf:${String(params?.i ?? "")}`
      return key
    },
  }),
}))

async function mountViewer() {
  const { default: PreviewImageViewerInner } = await import("./PreviewImageViewerInner.vue")
  return mount(PreviewImageViewerInner, {
    props: {
      images: ["https://example.test/1.jpg", "https://example.test/2.jpg"],
      initialIndex: 0,
      movieCode: "ABC-123",
    },
    global: {
      stubs: {
        DialogTitle: { template: '<h2 class="sr-only"><slot /></h2>' },
        DialogDescription: { template: '<p class="sr-only"><slot /></p>' },
      },
    },
  })
}

beforeEach(() => {
  carouselApi.selectedScrollSnap.mockReturnValue(0)
  vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => {
    cb(0)
    return 0
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  vi.clearAllMocks()
})

describe("PreviewImageViewerInner", () => {
  it("renders viewer chrome and image labels from locale keys", async () => {
    const wrapper = await mountViewer()

    expect(wrapper.text()).toContain("preview.title:ABC-123 1 / 2")
    expect(wrapper.text()).toContain("preview.instructions")

    expect(wrapper.find('button[aria-label="preview.close"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="preview.download"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="preview.previous"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="preview.next"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="preview.imageOf:1"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="preview.imageOf:2"]').exists()).toBe(true)

    const visibleAltTexts = wrapper
      .findAll("img")
      .map((img) => img.attributes("alt") ?? "")
      .filter((alt) => alt !== "")

    expect(visibleAltTexts).toEqual(["ABC-123 preview.imageOf:1", "ABC-123 preview.imageOf:2"])
  })

  it("tracks main image loading state and prioritizes only the selected full-size image", async () => {
    const wrapper = await mountViewer()
    const mainImages = wrapper
      .findAll("img")
      .filter((img) => (img.attributes("alt") ?? "") !== "")

    expect(mainImages[0]!.attributes("data-loaded")).toBe("false")
    expect(mainImages[0]!.attributes("loading")).toBe("eager")
    expect(mainImages[0]!.attributes("fetchpriority")).toBe("high")
    expect(mainImages[1]!.attributes("loading")).toBe("eager")
    expect(mainImages[1]!.attributes("fetchpriority")).toBe("low")

    await mainImages[0]!.trigger("load")

    expect(mainImages[0]!.attributes("data-loaded")).toBe("true")
  })

  it("downloads the selected preview image with a stable movie-code filename", async () => {
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {})
    const appendSpy = vi.spyOn(document.body, "appendChild")
    const wrapper = await mountViewer()

    carouselApi.selectedScrollSnap.mockReturnValue(1)
    const selectHandler = carouselApi.on.mock.calls.find(([event]) => event === "select")?.[1]
    expect(selectHandler).toBeTypeOf("function")
    ;(selectHandler as () => void)()
    await nextTick()

    await wrapper.get('button[aria-label="preview.download"]').trigger("click")

    const anchor = appendSpy.mock.calls.at(-1)?.[0] as HTMLAnchorElement | undefined
    expect(anchor?.tagName).toBe("A")
    expect(anchor?.href).toBe("https://example.test/2.jpg")
    expect(anchor?.download).toBe("ABC-123-preview-02.jpg")
    expect(anchor?.target).toBe("_blank")
    expect(anchor?.rel).toBe("noopener")
    expect(clickSpy).toHaveBeenCalledTimes(1)
  })
})
