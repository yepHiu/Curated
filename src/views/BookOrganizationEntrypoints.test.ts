import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { beforeEach, expect, it, vi } from "vitest"
import { i18n } from "@/i18n"
import ComicsView from "./ComicsView.vue"
import PhotosView from "./PhotosView.vue"

const api = vi.hoisted(() => ({ start: vi.fn(), ensure: vi.fn(async () => {}) }))
const books = ["a", "b", "c"].map(id => ({ id, title: id, sourceFileName: `${id}.zip`, tags: [], addedAt: "2026-10-17", updatedAt: "2026-10-17", pageCount: 1, currentPageIndex: 0, isFavorite: false, readStatus: "unread" }))
vi.mock("@/services/book-organization-service", () => ({ useBookOrganizationService: () => api }))
vi.mock("@/services/comic-library-service", () => ({ useComicLibraryService: () => ({ comics: ref(books), comicsLoaded: ref(true), loadError: ref(null), ensureComicsLoaded: api.ensure }) }))
vi.mock("@/services/photo-library-service", () => ({ usePhotoLibraryService: () => ({ photos: ref(books), photosLoaded: ref(true), loadError: ref(null), ensurePhotosLoaded: api.ensure }) }))
vi.mock("@/lib/experimental-agent", () => ({ useExperimentalAgent: () => ({ writeEnabled: ref(true) }) }))
vi.mock("@/composables/use-server-local-access", () => ({ useServerLocalAccess: () => ({ isServerLocal: ref(true) }) }))
vi.mock("@/composables/use-app-toast", () => ({ pushAppToast: vi.fn() }))
vi.mock("vue-router", () => ({ useRoute: () => ({ query: {}, fullPath: "/" }), useRouter: () => ({ push: vi.fn(), replace: vi.fn() }) }))
vi.mock("@/components/ui/dialog", () => ({ Dialog: { props: ["open"], template: '<section v-if="open" role="dialog"><slot /></section>' }, DialogClose: { template: '<span><slot /></span>' }, ...Object.fromEntries(["DialogContent", "DialogHeader", "DialogTitle", "DialogDescription", "DialogFooter"].map(name => [name, { template: '<div><slot /></div>' }])) }))
const page = { props: ["comics", "photos"], emits: ["enterBatchMode", "exitBatchMode", "toggleBatchSelect", "organize"], template: '<div />' }
beforeEach(() => { vi.clearAllMocks(); api.start.mockResolvedValue({ id: "job" }); i18n.global.locale.value = "zh-CN" })

it.each(["comic", "photo"] as const)("%s only submits selected books in batch mode, while a context action targets one book", async kind => {
  const wrapper = mount(kind === "comic" ? ComicsView : PhotosView, { global: { plugins: [i18n], stubs: { ComicLibraryPage: page, PhotoLibraryPage: page } } })
  await flushPromises()
  expect(wrapper.find('[role="toolbar"]').exists()).toBe(false)
  expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  const library = wrapper.findComponent(page)
  library.vm.$emit("enterBatchMode"); await flushPromises()
  const action = wrapper.get(`[data-${kind}-batch-organize]`)
  expect(action.attributes("disabled")).toBeDefined()
  library.vm.$emit("toggleBatchSelect", "b"); await flushPromises()
  await action.trigger("click")
  const confirm = () => wrapper.findAll("button").find(b => b.text() === "确认并开始")!
  await confirm().trigger("click"); await flushPromises()
  expect(api.start).toHaveBeenLastCalledWith(expect.objectContaining({ kind, bookIds: ["b"] }))
  library.vm.$emit("exitBatchMode"); await flushPromises()
  expect(wrapper.find('[role="toolbar"]').exists()).toBe(false)
  library.vm.$emit("organize", books[2]); await flushPromises()
  await confirm().trigger("click"); await flushPromises()
  expect(api.start).toHaveBeenLastCalledWith(expect.objectContaining({ kind, bookIds: ["c"] }))
  wrapper.unmount()
})
