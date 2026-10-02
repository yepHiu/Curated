// @vitest-environment jsdom
import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, expect, it, vi } from "vitest"
import { computed, ref } from "vue"
import type { TagOrganizationJob } from "@/services/contracts/topic-service"
import TagOrganizationDialog from "./TagOrganizationDialog.vue"
import SidebarTagOrganizationEntry from "./SidebarTagOrganizationEntry.vue"
import MovieLibraryContextMenu from "./MovieLibraryContextMenu.vue"
import type { Movie } from "@/domain/movie/types"

const state = vi.hoisted(() => ({ start: vi.fn(), cancel: vi.fn(), retry: vi.fn(), undo: vi.fn(), openSelected: vi.fn(), refreshStats: vi.fn(), refresh: vi.fn(), getItems: vi.fn() }))
const jobs = ref<TagOrganizationJob[]>([])
const stats = ref({ total: 10, organized: 4, unorganized: 5, outdated: 1, unresolved: 2 })
const statsLoading = ref(false)
const statsError = ref(false)
const open = ref(true)
const selection = ref<{ movieIds: string[]; title?: string } | null>(null)
vi.mock("@/composables/use-tag-organization", () => ({
  useTagOrganization: () => ({ ...state, jobs, selection, stats, statsLoading, statsError, active: computed(() => jobs.value.find((job) => job.status === "running")), dialogOpen: open, connected: ref(true), error: ref(""), busy: ref(false) }),
  isOrganizationActive: (job: TagOrganizationJob) => job.status === "running",
  organizationProgressText: () => "Processing 20/100",
  organizationProgressValue: () => 20,
  organizationErrorText: (code: string) => code,
}))
vi.mock("@/services/library-service", () => ({ useLibraryService: () => ({ listTopics: async () => [], setTopicHidden: vi.fn() }) }))
vi.mock("@/services/ai-service", () => ({ useAIService: () => ({ getTagOrganizationItems: state.getItems }) }))
vi.mock("@/lib/experimental-agent", () => ({ useExperimentalAgent: () => ({ writeEnabled: ref(true) }) }))
vi.mock("vue-i18n", () => ({ useI18n: () => ({ locale: ref("en"), t: (key: string, values?: Record<string, unknown>) => values ? `${key}:${JSON.stringify(values)}` : key }) }))
vi.mock("@/components/ui/dialog", () => ({
  Dialog: { template: "<div><slot /></div>" }, DialogContent: { template: "<div><slot /></div>" }, DialogHeader: { template: "<header><slot /></header>" }, DialogTitle: { template: "<h2><slot /></h2>" }, DialogDescription: { template: "<p><slot /></p>" },
}))
vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: { template: "<div><slot /></div>" }, DropdownMenuTrigger: { template: "<div><slot /></div>" }, DropdownMenuContent: { template: "<div><slot /></div>" }, DropdownMenuGroup: { template: "<div><slot /></div>" }, DropdownMenuItem: { template: "<button @click=\"$emit('select')\"><slot /></button>" },
}))

function historyJob(id: string, status: TagOrganizationJob["status"]): TagOrganizationJob {
 return { id, taskId: id, status, stage: "classifying", triggerReason: "manual", total: 100, processed: 20, succeeded: 20, unresolved: 0, failed: 0, revision: 1, createdAt: "2026-10-02T10:00:00Z", updatedAt: "2026-10-02T10:01:00Z" }
}
beforeEach(() => { vi.clearAllMocks(); statsLoading.value = false; statsError.value = false; stats.value = { total: 10, organized: 4, unorganized: 5, outdated: 1, unresolved: 2 }; open.value = true; selection.value = null; jobs.value = []; state.getItems.mockResolvedValue([]) })

it("shows source-aware coverage and starts only the chosen remaining scope", async () => {
 const wrapper = mount(TagOrganizationDialog)
 expect(wrapper.get('[data-organization-count="organized"]').text()).toBe("4")
 expect(wrapper.get('[data-organization-count="unorganized"]').text()).toBe("5")
 expect(wrapper.get('[data-organization-count="outdated"]').text()).toBe("1")
 expect(wrapper.text()).toContain('topics.analyzedWithoutMatch:{"count":2}')
 expect(state.start).not.toHaveBeenCalled()
 await wrapper.get('[data-organize-unorganized]').trigger('click')
 expect(state.start).toHaveBeenLastCalledWith()
 await wrapper.get('[data-organize-outdated]').trigger('click')
 expect(state.start).toHaveBeenLastCalledWith('outdated')
 statsError.value = true
 await flushPromises()
 expect(wrapper.get('[data-organize-unorganized]').attributes('disabled')).toBeDefined()
 expect(wrapper.text()).toContain('topics.coverageFailed')
 wrapper.unmount()
})

it("disables remaining work when every movie is already current", async () => {
 stats.value = { total: 10, organized: 10, unorganized: 0, outdated: 0, unresolved: 2 }
 const wrapper = mount(TagOrganizationDialog)
 expect(wrapper.get('[data-organize-unorganized]').attributes('disabled')).toBeDefined()
 expect(wrapper.find('[data-organize-outdated]').exists()).toBe(false)
 expect(wrapper.text()).toContain('topics.allOrganized')
 wrapper.unmount()
})

it("renders compact history rows and reads evidence only when requested", async () => {
 jobs.value = [historyJob("one", "completed"), historyJob("two", "blocked")]
 const wrapper = mount(TagOrganizationDialog)
 expect(wrapper.findAll("[data-organization-history-row]")).toHaveLength(2)
 expect(wrapper.get("[data-organization-history]").classes()).toContain("max-h-60")
 expect(wrapper.findAll("time")[0]!.attributes("datetime")).toBe("2026-10-02T10:00:00Z")
 expect(wrapper.text()).toContain('topics.movieCount:{"count":100}')
 expect(wrapper.find("[data-organization-history-detail]").exists()).toBe(false)
 await wrapper.get("button[aria-expanded]").trigger("click")
 expect(wrapper.find("[data-organization-history-detail]").exists()).toBe(true)
 expect(state.getItems).not.toHaveBeenCalled()
 const details = wrapper.findAll("button").find((button) => button.text() === "topics.details")!
 await details.trigger("click"); await flushPromises()
 expect(state.getItems).toHaveBeenCalledWith("one", 0)
 wrapper.unmount()
})

it("previews the captured single-movie scope and starts only on click", async () => {
 selection.value = { movieIds: ["one"], title: "Selected movie" }
 const wrapper = mount(TagOrganizationDialog)
 expect(wrapper.get("[data-organization-scope]").text()).toContain("Selected movie")
 expect(state.start).not.toHaveBeenCalled()
 await wrapper.get("[data-organization-scope] button").trigger("click")
 expect(state.start).toHaveBeenCalledOnce()
 wrapper.unmount()
})

it("shows a dedicated sidebar progress entry in normal and compact modes", async () => {
 jobs.value = [historyJob("one", "running")]
 const wrapper = mount(SidebarTagOrganizationEntry)
 expect(wrapper.get("[data-slot='progress']").attributes("aria-valuenow")).toBe("20")
 expect(wrapper.text()).toContain("Processing 20/100")
 expect(wrapper.classes()).not.toContain("fixed")
 open.value = false
 await wrapper.get("button").trigger("click")
 expect(open.value).toBe(true)
 await wrapper.setProps({ compact: true })
 expect(wrapper.get("button").attributes("aria-label")).toContain("Processing 20/100")
 wrapper.unmount()
})

it("opens a selected-movie scope from the right-click menu and hides it for trash", async () => {
 const movie = { id: "one", title: "A movie", location: "", trashedAt: "" } as Movie
 const wrapper = mount(MovieLibraryContextMenu, { props: { movie, x: 10, y: 10 } })
 const menuItem = document.querySelector<HTMLButtonElement>("[data-ai-organize-movie]")!
 menuItem.click(); await flushPromises()
 expect(state.openSelected).toHaveBeenCalledWith(["one"], "A movie")
 expect(wrapper.emitted("close")).toBeTruthy()
 await wrapper.setProps({ movie: { ...movie, trashedAt: "2026-10-02" } })
 expect(document.querySelector("[data-ai-organize-movie]")).toBeNull()
 wrapper.unmount()
})
