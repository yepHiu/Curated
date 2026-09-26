import { flushPromises, mount, DOMWrapper } from "@vue/test-utils"
import { afterEach, describe, expect, it, vi } from "vitest"
import SettingsServerAddDialog from "./SettingsServerAddDialog.vue"

vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const wrappers: ReturnType<typeof mount>[] = []
async function setup() {
  const addServer = vi.fn().mockResolvedValue(undefined)
  const openServerConnections = vi.fn()
  const getServerConnections = vi.fn().mockResolvedValue({ servers: [], connecting: false })
  window.javLibrary = { addServer, openServerConnections, getServerConnections }
  const wrapper = mount(SettingsServerAddDialog, { attachTo: document.body })
  wrappers.push(wrapper)
  await wrapper.get('button').trigger('click')
  await flushPromises()
  const body = new DOMWrapper(document.body)
  await body.get('#new-server-name').setValue(' 家庭 NAS ')
  await body.get('#new-server-url').setValue('NAS.local:8081/')
  return { wrapper, body, addServer, openServerConnections, getServerConnections }
}
afterEach(() => { wrappers.splice(0).forEach(wrapper => wrapper.unmount()); document.body.innerHTML = ''; delete window.javLibrary })

describe('Add server dialog', () => {
  it('adds from an in-page dialog, normalizes input, closes and refreshes without opening a window', async () => {
    const { wrapper, body, addServer, openServerConnections } = await setup()
    expect(body.find('[role="dialog"]').exists()).toBe(true)
    await body.get('form').trigger('submit')
    await flushPromises()
    expect(addServer).toHaveBeenCalledWith({ name: '家庭 NAS', url: 'http://nas.local:8081' })
    expect(wrapper.emitted('saved')).toHaveLength(1)
    expect(openServerConnections).not.toHaveBeenCalled()
    expect(body.find('[role="dialog"]').exists()).toBe(false)
  })
  it('rejects non-root addresses without sending a write', async () => {
    const { body, addServer } = await setup()
    await body.get('#new-server-url').setValue('https://nas.local/library')
    await body.get('form').trigger('submit')
    await flushPromises()
    expect(body.text()).toContain('serverConnections.invalidAddress')
    expect(addServer).not.toHaveBeenCalled()
  })
  it('does not overwrite duplicate saved servers', async () => {
    const { body, getServerConnections, addServer } = await setup()
    getServerConnections.mockResolvedValue({ servers: [{ url: 'http://nas.local:8081' }], connecting: false })
    await body.get('form').trigger('submit')
    await flushPromises()
    expect(body.text()).toContain('serverConnections.duplicateAddress')
    expect(addServer).not.toHaveBeenCalled()
  })
  it('retains drafts and allows retry after a failed save', async () => {
    const { body, addServer } = await setup()
    addServer.mockRejectedValueOnce(new Error('write failed'))
    await body.get('form').trigger('submit')
    await flushPromises()
    expect(body.text()).toContain('serverConnections.saveError')
    expect((body.get('#new-server-name').element as HTMLInputElement).value).toBe(' 家庭 NAS ')
    expect((body.get('#new-server-url').element as HTMLInputElement).value).toBe('NAS.local:8081/')
    await body.get('form').trigger('submit')
    await flushPromises()
    expect(addServer).toHaveBeenCalledTimes(2)
  })
  it('blocks duplicate submits and dismissals while saving', async () => {
    const { body, addServer } = await setup()
    let finish = () => {}
    addServer.mockImplementation(() => new Promise<void>(resolve => { finish = resolve }))
    await body.get('form').trigger('submit')
    await flushPromises()
    await body.get('form').trigger('submit')
    expect(addServer).toHaveBeenCalledOnce()
    expect(body.get('[data-save-server]').attributes('disabled')).toBeDefined()
    expect(body.find('[data-slot="dialog-close"]:not([disabled])').exists()).toBe(false)
    await body.get('[role="dialog"]').trigger('keydown', { key: 'Escape' })
    expect(body.find('[role="dialog"]').exists()).toBe(true)
    finish()
    await flushPromises()
  })
  it('explains that old Desktop main processes require a full restart', async () => {
    const { body, addServer } = await setup()
    addServer.mockRejectedValue(new Error("No handler registered for 'curated:add-server'"))
    await body.get('form').trigger('submit')
    await flushPromises()
    expect(body.text()).toContain('serverConnections.restartRequired')
  })
})
