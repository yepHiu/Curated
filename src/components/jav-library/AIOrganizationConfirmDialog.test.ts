// @vitest-environment jsdom
import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, expect, it, vi } from "vitest"
import { ref } from "vue"
import AIOrganizationConfirmDialog from "./AIOrganizationConfirmDialog.vue"

const start = vi.hoisted(() => vi.fn())
const open = ref(true), busy = ref(false), error = ref(""), writeEnabled = ref(true)
const selection = ref({ movieIds: ["one", "two"], movies: [{ id: "one", title: "First movie", code: "CODE-001" }, { id: "two", title: "Second movie", code: "CODE-002" }] })
vi.mock("@/composables/use-tag-organization", () => ({ useTagOrganization: () => ({ dialogOpen: open, busy, error, selection, startSelected: start }) }))
vi.mock("@/lib/experimental-agent", () => ({ useExperimentalAgent: () => ({ writeEnabled }) }))
vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock("@/components/ui/dialog", () => ({
  Dialog: { template: '<div><slot /></div>' }, DialogContent: { template: '<div><slot /></div>' }, DialogHeader: { template: '<header><slot /></header>' }, DialogTitle: { template: '<h2><slot /></h2>' }, DialogDescription: { template: '<p><slot /></p>' }, DialogFooter: { template: '<footer><slot /></footer>' },
}))
beforeEach(() => { vi.clearAllMocks(); open.value = true; busy.value = false; error.value = ""; writeEnabled.value = true })

it("confirms multiple identities and sends only the checked actions", async () => {
 const w = mount(AIOrganizationConfirmDialog)
 expect(w.get('[data-organization-movies]').text()).toContain('First movie')
 expect(w.get('[data-organization-movies]').text()).toContain('CODE-002')
 expect(start).not.toHaveBeenCalled()
 await w.get('[data-organization-task="tags"]').trigger('click')
 expect(w.get('[data-confirm-organization]').attributes('disabled')).toBeDefined()
 await w.get('[data-organization-task="title"]').trigger('click')
 await w.get('[data-organization-task="summary"]').trigger('click')
 await w.get('[data-confirm-organization]').trigger('click')
 expect(start).toHaveBeenCalledWith(['title', 'summary'])
 open.value = false; await flushPromises(); open.value = true; await flushPromises()
 expect(w.get('[data-organization-task="tags"]').attributes('data-state')).toBe('checked')
 expect(w.get('[data-organization-task="title"]').attributes('data-state')).toBe('unchecked')
 w.unmount()
})

it("keeps selections and errors on failure, and disables duplicate submission and readonly writes", async () => {
 const w = mount(AIOrganizationConfirmDialog)
 await w.get('[data-organization-task="title"]').trigger('click')
 error.value = 'Submission failed'; await flushPromises()
 expect(w.text()).toContain('Submission failed')
 expect(w.get('[data-organization-task="title"]').attributes('data-state')).toBe('checked')
 busy.value = true; await flushPromises()
 expect(w.get('[data-confirm-organization]').attributes('disabled')).toBeDefined()
 expect(w.get('[data-organization-task="title"]').attributes('disabled')).toBeDefined()
 busy.value = false; writeEnabled.value = false; await flushPromises()
 expect(w.get('[data-confirm-organization]').attributes('disabled')).toBeDefined()
 w.unmount()
})

it("dismisses without creating tasks", async () => {
 const w = mount(AIOrganizationConfirmDialog)
 await w.findAll('button').find(button => button.text() === 'common.cancel')!.trigger('click')
 expect(open.value).toBe(false); expect(start).not.toHaveBeenCalled()
 w.unmount()
})
