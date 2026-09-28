const access = vi.hoisted(() => ({ value: true }))
vi.mock("@/composables/use-server-local-access", () => ({ useServerLocalAccess: () => ({ isServerLocal: computed(() => access.value) }) }))
import { computed, nextTick, ref } from "vue"
import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"

const backendLogState = ref({
  logDir: "",
  logLevel: "info",
} as { logDir?: string; logFilePrefix?: string; logMaxAgeDays?: number; logLevel?: string })

const patchBackendLog = vi.fn()
const revealLogDirectory = vi.fn()

vi.mock("@/components/ui/badge", () => ({
  Badge: { name: "Badge", template: "<span><slot /></span>" },
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  Activity: { name: "Activity", template: "<span />" },
  FolderOpen: { name: "FolderOpen", template: "<span />" },
  ScrollText: { name: "ScrollText", template: "<span />" },
}))

vi.mock("@/api/http-client", () => ({
  HttpClientError: class HttpClientError extends Error {},
}))

vi.mock("@/composables/use-settings-scroll-preserve", () => ({
  useSettingsScrollPreserve: () => ({
    withPreservedScroll: async <T>(fn: () => Promise<T> | T) => await fn(),
  }),
}))

vi.mock("@/lib/app-logger", () => ({
  CLIENT_LOG_LEVEL_OPTIONS: ["trace", "debug", "info", "warn", "error"],
  getClientLogLevelName: () => "info",
  setClientLogLevel: vi.fn(),
}))

vi.mock("@/lib/pick-directory", () => ({
  pickLibraryDirectory: vi.fn(),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    backendLog: computed(() => backendLogState.value),
    patchBackendLog,
    revealLogDirectory,
  }),
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    emits: ["click"],
    template: "<button @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/card", () => ({
  Card: { name: "Card", template: "<div><slot /></div>" },
  CardContent: { name: "CardContent", template: "<div><slot /></div>" },
  CardDescription: { name: "CardDescription", template: "<div><slot /></div>" },
  CardHeader: { name: "CardHeader", template: "<div><slot /></div>" },
  CardTitle: { name: "CardTitle", template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/input", () => ({
  Input: {
    name: "Input",
    props: ["modelValue"],
    emits: ["update:modelValue", "input"],
    template:
      "<input :value=\"modelValue\" @input=\"$emit('update:modelValue', $event.target.value); $emit('input', $event)\" />",
  },
}))

vi.mock("@/components/ui/select", () => {
  const Select = {
    name: "Select",
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template: "<div class=\"select-stub\" :data-model-value=\"modelValue\"><slot /></div>",
  }
  return {
    Select,
    SelectContent: { name: "SelectContent", template: "<div><slot /></div>" },
    SelectItem: { name: "SelectItem", props: ["value"], template: "<div><slot /></div>" },
    SelectTrigger: { name: "SelectTrigger", template: "<div><slot /></div>" },
    SelectValue: { name: "SelectValue", template: "<div><slot /></div>" },
  }
})

async function mountComponent(autoSaveReady = false) {
  vi.resetModules()
  vi.stubEnv("VITE_USE_WEB_API", "true")
  const mod = await import("./SettingsLoggingSection.vue")
  return mount(mod.default, {
    props: {
      autoSaveReady,
    },
  })
}

function backendLogLevelSelectValue(wrapper: ReturnType<typeof mount>) {
  return wrapper.findAll(".select-stub")[1]?.attributes("data-model-value")
}

describe("SettingsLoggingSection", () => {
  beforeEach(() => {
    access.value = true
    delete window.javLibrary
    revealLogDirectory.mockReset().mockResolvedValue(undefined)
    backendLogState.value = {
      logDir: "",
      logLevel: "info",
    }
    patchBackendLog.mockReset().mockResolvedValue(undefined)
  })

  it("syncs backend log drafts from service after autoSaveReady becomes true", async () => {
    const wrapper = await mountComponent(false)

    expect(backendLogLevelSelectValue(wrapper)).toBe("info")

    backendLogState.value = {
      logDir: "D:/logs",
      logLevel: "debug",
    }
    await wrapper.setProps({ autoSaveReady: true })
    await nextTick()
    await flushPromises()

    expect(backendLogLevelSelectValue(wrapper)).toBe("debug")
    wrapper.unmount()
  })
})

it("hides backend log configuration remotely", async () => {
  access.value = false
  const wrapper = await mountComponent(true)
  expect(wrapper.text()).not.toContain("settings.backendLogTitle")
  wrapper.unmount()
})

it("opens the Server directory only from local Desktop and reports failures", async () => {
  access.value = true
  Object.defineProperty(window, "javLibrary", { configurable: true, value: {} })
  revealLogDirectory.mockReset().mockRejectedValueOnce(new Error("failed")).mockResolvedValue(undefined)
  const wrapper = await mountComponent(true)
  expect(wrapper.find("input").exists()).toBe(false)
  const button = wrapper.findAll("button").find(b => b.text() === "settings.backendLogOpenDirectory")!
  await button.trigger("click")
  await flushPromises()
  expect(revealLogDirectory).toHaveBeenCalledTimes(1)
  expect(revealLogDirectory).toHaveBeenCalledWith()
  expect(wrapper.text()).toContain("settings.backendLogOpenFailed")
  await button.trigger("click")
  await flushPromises()
  expect(wrapper.text()).not.toContain("settings.backendLogOpenFailed")
  wrapper.unmount()
  delete window.javLibrary
})

it("keeps local Web log settings but hides directory opening", async () => {
  access.value = true
  delete window.javLibrary
  const wrapper = await mountComponent(true)
  expect(wrapper.text()).toContain("settings.backendLogMaxAge")
  expect(wrapper.text()).toContain("settings.backendLogLevel")
  expect(wrapper.text()).not.toContain("settings.backendLogOpenDirectory")
  wrapper.unmount()
})

it("hides Server controls from remote Desktop while keeping client logs", async () => {
  access.value = false
  Object.defineProperty(window, "javLibrary", { configurable: true, value: {} })
  const wrapper = await mountComponent(true)
  expect(wrapper.text()).not.toContain("settings.backendLogOpenDirectory")
  expect(wrapper.text()).not.toContain("settings.backendLogLevel")
  expect(wrapper.text()).toContain("settings.clientLogLevel")
  wrapper.unmount()
  delete window.javLibrary
})

it("saves retention and level without sending a directory override", async () => {
  access.value = true
  patchBackendLog.mockReset().mockImplementation(async patch => {
    backendLogState.value = { ...backendLogState.value, ...patch }
  })
  const wrapper = await mountComponent(true)
  wrapper.findAllComponents({ name: "Select" })[0]!.vm.$emit("update:modelValue", "10")
  wrapper.findAllComponents({ name: "Select" })[1]!.vm.$emit("update:modelValue", "debug")
  await nextTick()
  await new Promise(resolve => setTimeout(resolve, 650))
  await flushPromises()
  expect(patchBackendLog).toHaveBeenCalledWith({ logMaxAgeDays: 10, logLevel: "debug" })
  wrapper.unmount()
})
