import { computed, nextTick, ref } from "vue"
import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"

const backendLogState = ref({
  logDir: "",
  logLevel: "info",
} as { logDir?: string; logFilePrefix?: string; logMaxAgeDays?: number; logLevel?: string })

const patchBackendLog = vi.fn()

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("@vueuse/core", () => ({
  watchDebounced: (
    source: unknown,
    cb: unknown,
  ) => {
    void source
    void cb
    return () => {}
  },
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
    backendLogState.value = {
      logDir: "",
      logLevel: "info",
    }
    patchBackendLog.mockReset()
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
  })
})
