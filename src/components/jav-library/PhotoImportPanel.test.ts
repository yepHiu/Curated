import { flushPromises, mount } from "@vue/test-utils"
import { computed } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import PhotoImportPanel from "./PhotoImportPanel.vue"
const mocks = vi.hoisted(() => ({ enabled: true, target: "photos", importPhotos: vi.fn(), refreshSettings: vi.fn(), start: vi.fn(), toast: vi.fn() }))
vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock("@/services/photo-library-service", () => ({ usePhotoLibraryService: () => ({ photoLibraryEnabled: computed(() => mocks.enabled), photoLibraryPaths: computed(() => [{id: "photos", path: "D:/Photos"}]), defaultPhotoImportLibraryPathId: computed(() => mocks.target), importPhotos: mocks.importPhotos, refreshSettings: mocks.refreshSettings }) }))
vi.mock("@/composables/use-scan-task-tracker", () => ({ useScanTaskTracker: () => ({ start: mocks.start }) }))
vi.mock("@/composables/use-app-toast", () => ({ pushAppToast: mocks.toast }))
beforeEach(() => { vi.clearAllMocks(); mocks.enabled = true; mocks.target = "photos"; mocks.refreshSettings.mockResolvedValue(undefined) })
async function select(wrapper: ReturnType<typeof mount>, files = [new File(["zip"], "photos.zip")]) {
  const input = wrapper.get<HTMLInputElement>("[data-photo-import-input]")
  Object.defineProperty(input.element, "files", { value: files, configurable: true })
  await input.trigger("change")
}
describe("PhotoImportPanel", () => {
  it.each([[false, "photos"], [true, ""]])("cannot upload without enabled Beta and default target", async (enabled, target) => {
    mocks.enabled = enabled; mocks.target = target
    const wrapper = mount(PhotoImportPanel, { props: { active: true } })
    await select(wrapper)
    expect(wrapper.get<HTMLButtonElement>("[data-photo-import-submit]").element.disabled).toBe(true)
    expect(mocks.importPhotos).not.toHaveBeenCalled()
    wrapper.unmount()
  })
  it("filters unsupported files and follows the independent photo scan after upload", async () => {
    mocks.importPhotos.mockImplementation(async (_files, options) => { options.onUploadProgress({ loaded: 3, total: 3, percent: 100 }); return {status: "completed", metadata: {scanTaskId: "photo-scan"}} })
    const wrapper = mount(PhotoImportPanel, { props: { active: true } })
    const zip = new File(["zip"], "photos.zip")
    await select(wrapper, [zip, zip, new File(["rar"], "unsupported.rar")])
    expect(wrapper.text()).toContain("import.photoSkippedUnsupported")
    await wrapper.get("[data-photo-import-submit]").trigger("click"); await flushPromises()
    expect(mocks.importPhotos).toHaveBeenCalledWith([zip], {onUploadProgress: expect.any(Function)})
    expect(mocks.start).toHaveBeenCalledWith("photo-scan")
    expect(wrapper.emitted("busy")).toEqual([[true], [false]])
    expect(wrapper.emitted("completed")).toHaveLength(1)
    wrapper.unmount()
  })
  it.each(["partial_failed", "failed"])("keeps files and reports %s instead of closing with success", async status => {
    mocks.importPhotos.mockResolvedValue({status, metadata: {completedFiles: 1, failedFiles: 1}})
    const wrapper = mount(PhotoImportPanel, { props: { active: true } })
    await select(wrapper)
    await wrapper.get("[data-photo-import-submit]").trigger("click"); await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toBe("import.photoResultError")
    expect(wrapper.text()).toContain("photos.zip")
    expect(wrapper.emitted("completed")).toBeUndefined()
    expect(mocks.toast).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
