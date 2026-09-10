import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import ImportMenu from "./ImportMenu.vue"

const state = vi.hoisted(() => ({ comic: undefined as unknown, photo: undefined as unknown, refresh: vi.fn() }))
vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock("@/services/comic-library-service", () => ({ useComicLibraryService: () => ({ comicLibraryEnabled: state.comic, refreshSettings: state.refresh }) }))
vi.mock("@/services/photo-library-service", () => ({ usePhotoLibraryService: () => ({ photoLibraryEnabled: state.photo, refreshSettings: state.refresh }) }))
vi.mock("./MovieImportDialog.vue", () => ({ __esModule: true, default: { name: "MovieImportDialog", props: ["open", "embedded"], emits: ["busy", "completed"], template: '<input data-movie-files />' } }))
vi.mock("./ComicImportDialog.vue", () => ({ __esModule: true, default: { name: "ComicImportDialog", props: ["open", "embedded"], emits: ["busy", "completed"], template: '<input data-comic-files />' } }))
vi.mock("./PhotoImportPanel.vue", () => ({ __esModule: true, default: { name: "PhotoImportPanel", props: ["active"], emits: ["busy", "completed"], template: '<input data-photo-files />' } }))

const wrappers: ReturnType<typeof mount>[] = []
beforeEach(() => { state.comic = ref(false); state.photo = ref(false); state.refresh.mockResolvedValue(undefined) })
afterEach(() => { wrappers.forEach(wrapper => wrapper.unmount()); wrappers.length = 0; document.body.innerHTML = "" })
async function openMenu(comic: boolean, photo: boolean) {
  (state.comic as ReturnType<typeof ref>).value = comic
  ;(state.photo as ReturnType<typeof ref>).value = photo
  const wrapper = mount(ImportMenu, { attachTo: document.body })
  wrappers.push(wrapper)
  await wrapper.get("[data-import-trigger]").trigger("click")
  await flushPromises()
  return wrapper
}

describe("unified media import", () => {
  it.each([[false, false], [true, false], [false, true], [true, true]])("shows only enabled tabs: comic=%s photo=%s", async (comic, photo) => {
    const wrapper = await openMenu(comic, photo)
    expect(wrapper.get("[data-import-trigger]").text()).toBe("import.mediaTrigger")
    expect(document.querySelectorAll('[role="dialog"]')).toHaveLength(1)
    const tabs = Array.from(document.querySelectorAll('[role="tab"]')).map(tab => tab.textContent)
    expect(tabs).toEqual(["import.trigger", ...(comic ? ["import.comicTrigger"] : []), ...(photo ? ["import.photoTrigger"] : [])])
    expect(document.querySelector('[role="tab"][data-state="active"]')?.textContent).toBe("import.trigger")
  })

  it("preserves panel state on switching and returns to movies when a gate closes", async () => {
    await openMenu(true, true)
    const photoTab = Array.from(document.querySelectorAll<HTMLButtonElement>('[role="tab"]')).find(tab => tab.textContent === "import.photoTrigger")!
    photoTab.click(); await flushPromises()
    const input = document.querySelector<HTMLInputElement>("[data-photo-files]")!
    input.value = "selected archive"
    document.querySelector<HTMLButtonElement>('[role="tab"]')!.click(); await flushPromises()
    photoTab.click(); await flushPromises()
    expect(document.querySelector<HTMLInputElement>("[data-photo-files]")!.value).toBe("selected archive")
    ;(state.photo as ReturnType<typeof ref>).value = false
    await flushPromises()
    expect(document.querySelector('[role="tab"][data-state="active"]')?.textContent).toBe("import.trigger")
    expect(document.querySelector('[data-photo-files]')).toBeNull()
  })

  it("blocks tab changes and closing during an upload, then closes on completion", async () => {
    const wrapper = await openMenu(true, true)
    const movie = wrapper.findComponent({ name: "MovieImportDialog" })
    movie.vm.$emit("busy", true); await flushPromises()
    expect(Array.from(document.querySelectorAll<HTMLButtonElement>('[role="tab"]')).every(tab => tab.disabled)).toBe(true)
    wrapper.findComponent({ name: "Dialog" }).vm.$emit("update:open", false)
    await flushPromises()
    expect(document.querySelector('[role="dialog"]')).not.toBeNull()
    movie.vm.$emit("completed"); await flushPromises()
    expect(document.querySelector('[role="dialog"]')).toBeNull()
  })
})
