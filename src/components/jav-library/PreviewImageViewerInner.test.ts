import { mount } from "@vue/test-utils"
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
  vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => {
    cb(0)
    return 0
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

describe("PreviewImageViewerInner", () => {
  it("renders viewer chrome and image labels from locale keys", async () => {
    const wrapper = await mountViewer()

    expect(wrapper.text()).toContain("preview.title:ABC-123 1 / 2")
    expect(wrapper.text()).toContain("preview.instructions")

    expect(wrapper.find('button[aria-label="preview.close"]').exists()).toBe(true)
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
})
