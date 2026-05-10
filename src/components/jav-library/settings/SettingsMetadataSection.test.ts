import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsMetadataSection from "./SettingsMetadataSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  Sparkles: { name: "Sparkles", template: "<span />" },
}))

vi.mock("@/components/ui/card", () => ({
  Card: { name: "Card", template: "<section><slot /></section>" },
  CardContent: { name: "CardContent", template: "<div><slot /></div>" },
  CardHeader: { name: "CardHeader", template: "<header><slot /></header>" },
  CardTitle: { name: "CardTitle", template: "<h3><slot /></h3>" },
}))

vi.mock("./SettingsMetadataAutomationSection.vue", () => ({
  default: {
    name: "SettingsMetadataAutomationSection",
    props: [
      "useWebApi",
      "providerPingAllBusy",
      "providerPingOneName",
      "providerHealthPingAllSummary",
      "providerHealthPingError",
      "autoLibraryWatch",
      "autoLibraryWatchSaving",
      "autoLibraryWatchError",
      "autoActorProfileScrape",
      "autoActorProfileScrapeSaving",
      "autoActorProfileScrapeError",
    ],
    emits: ["pingAllProviders", "changeAutoLibraryWatch", "changeAutoActorProfileScrape"],
    template:
      "<div data-automation><button data-ping-all @click=\"$emit('pingAllProviders')\">ping</button><button data-auto-watch @click=\"$emit('changeAutoLibraryWatch', true)\">watch</button><button data-auto-actor @click=\"$emit('changeAutoActorProfileScrape', true)\">actor</button></div>",
  },
}))

vi.mock("./SettingsMetadataModeSection.vue", () => ({
  default: {
    name: "SettingsMetadataModeSection",
    props: [
      "metadataMovieModeUi",
      "metadataMovieSaving",
      "metadataMovieChainSaving",
      "providerPingAllBusy",
      "canPickSpecifiedMetadata",
      "canUseMetadataChainMode",
    ],
    emits: ["selectAuto", "selectSpecified", "selectChain"],
    template:
      "<div data-mode><button data-select-auto @click=\"$emit('selectAuto')\">auto</button><button data-select-specified @click=\"$emit('selectSpecified')\">specified</button><button data-select-chain @click=\"$emit('selectChain')\">chain</button></div>",
  },
}))

vi.mock("./SettingsMetadataProviderSelectSection.vue", () => ({
  default: {
    name: "SettingsMetadataProviderSelectSection",
    props: [
      "useWebApi",
      "metadataMovieProvider",
      "metadataMovieSelectOptions",
      "metadataMovieSaving",
      "providerPingAllBusy",
      "providerPingOneName",
      "currentProviderHealth",
    ],
    emits: ["selectProvider", "pingProvider"],
    template:
      "<div data-provider-select><span data-current-health>{{ currentProviderHealth?.status || 'none' }}</span><button data-select-provider @click=\"$emit('selectProvider', 'JavBus')\">select</button><button data-ping-provider @click=\"$emit('pingProvider', metadataMovieProvider)\">ping one</button></div>",
  },
}))

vi.mock("./SettingsMetadataProviderChainSection.vue", () => ({
  default: {
    name: "SettingsMetadataProviderChainSection",
    props: [
      "useWebApi",
      "canPickSpecifiedMetadata",
      "providerChainDraft",
      "availableProvidersForChain",
      "selectedProviderToAdd",
      "chainDragFromIndex",
      "metadataMovieChainSaving",
      "metadataMovieChainError",
      "providerPingAllBusy",
      "providerPingOneName",
      "providerHealthByName",
    ],
    emits: [
      "dragStart",
      "dragOver",
      "dropProvider",
      "dragEnd",
      "pingProvider",
      "removeProvider",
      "update:selectedProviderToAdd",
      "addProvider",
      "saveProviderChain",
    ],
    template:
      "<div data-chain><button data-drag-start @click=\"$emit('dragStart', $event, 1)\">drag start</button><button data-drag-over @click=\"$emit('dragOver', $event)\">drag over</button><button data-drop @click=\"$emit('dropProvider', 1)\">drop</button><button data-drag-end @click=\"$emit('dragEnd')\">drag end</button><button data-chain-ping @click=\"$emit('pingProvider', providerChainDraft[0])\">ping</button><button data-chain-remove @click=\"$emit('removeProvider', 0)\">remove</button><button data-chain-select-add @click=\"$emit('update:selectedProviderToAdd', 'JavBus')\">select add</button><button data-chain-add @click=\"$emit('addProvider')\">add</button><button data-chain-save @click=\"$emit('saveProviderChain')\">save</button></div>",
  },
}))

vi.mock("./SettingsMetadataTriggerScrapeSection.vue", () => ({
  default: {
    name: "SettingsMetadataTriggerScrapeSection",
    props: ["busy", "success", "error"],
    emits: ["run"],
    template:
      "<div data-trigger><button data-trigger-run @click=\"$emit('run')\">run</button></div>",
  },
}))

const providerHealth = {
  name: "javdb",
  status: "ok" as const,
  latencyMs: 42,
}

const baseProps = {
  useWebApi: true,
  providerPingAllBusy: false,
  providerPingOneName: null,
  providerHealthPingAllSummary: "",
  providerHealthPingError: "",
  autoLibraryWatch: false,
  autoLibraryWatchSaving: false,
  autoLibraryWatchError: "",
  autoActorProfileScrape: false,
  autoActorProfileScrapeSaving: false,
  autoActorProfileScrapeError: "",
  metadataMovieModeUi: "specified" as const,
  metadataMovieSaving: false,
  metadataMovieChainSaving: false,
  canPickSpecifiedMetadata: true,
  canUseMetadataChainMode: true,
  metadataMovieProvider: "JavDB",
  metadataMovieSelectOptions: ["JavDB", "JavBus"],
  metadataMovieError: "",
  providerChainDraft: ["JavDB"],
  availableProvidersForChain: ["JavBus"],
  selectedProviderToAdd: "",
  chainDragFromIndex: null,
  metadataMovieChainError: "",
  providerHealthByName: {
    javdb: providerHealth,
  },
  triggerScrapeCardBusy: false,
  triggerScrapeCardSuccess: "",
  triggerScrapeCardError: "",
}

describe("SettingsMetadataSection", () => {
  it("renders metadata card copy and specified provider health", () => {
    const wrapper = mount(SettingsMetadataSection, {
      props: baseProps,
    })

    expect(wrapper.text()).toContain("settings.metadataMovieProviderTitle")
    expect(wrapper.find("[data-provider-select]").exists()).toBe(true)
    expect(wrapper.find("[data-chain]").exists()).toBe(false)
    expect(wrapper.get("[data-current-health]").text()).toBe("ok")
  })

  it("renders chain mode and forwards metadata action events", async () => {
    const wrapper = mount(SettingsMetadataSection, {
      props: {
        ...baseProps,
        metadataMovieModeUi: "chain",
      },
    })

    await wrapper.get("[data-ping-all]").trigger("click")
    await wrapper.get("[data-auto-watch]").trigger("click")
    await wrapper.get("[data-auto-actor]").trigger("click")
    await wrapper.get("[data-select-auto]").trigger("click")
    await wrapper.get("[data-select-specified]").trigger("click")
    await wrapper.get("[data-select-chain]").trigger("click")
    await wrapper.get("[data-drag-start]").trigger("click")
    await wrapper.get("[data-drag-over]").trigger("click")
    await wrapper.get("[data-drop]").trigger("click")
    await wrapper.get("[data-drag-end]").trigger("click")
    await wrapper.get("[data-chain-ping]").trigger("click")
    await wrapper.get("[data-chain-remove]").trigger("click")
    await wrapper.get("[data-chain-select-add]").trigger("click")
    await wrapper.get("[data-chain-add]").trigger("click")
    await wrapper.get("[data-chain-save]").trigger("click")
    await wrapper.get("[data-trigger-run]").trigger("click")

    expect(wrapper.emitted("pingAllProviders")).toHaveLength(1)
    expect(wrapper.emitted("changeAutoLibraryWatch")).toEqual([[true]])
    expect(wrapper.emitted("changeAutoActorProfileScrape")).toEqual([[true]])
    expect(wrapper.emitted("selectAuto")).toHaveLength(1)
    expect(wrapper.emitted("selectSpecified")).toHaveLength(1)
    expect(wrapper.emitted("selectChain")).toHaveLength(1)
    expect(wrapper.emitted("dragStart")?.[0]?.[1]).toBe(1)
    expect(wrapper.emitted("dragOver")).toHaveLength(1)
    expect(wrapper.emitted("dropProvider")).toEqual([[1]])
    expect(wrapper.emitted("dragEnd")).toHaveLength(1)
    expect(wrapper.emitted("pingProvider")).toEqual([["JavDB"]])
    expect(wrapper.emitted("removeProvider")).toEqual([[0]])
    expect(wrapper.emitted("update:selectedProviderToAdd")).toEqual([["JavBus"]])
    expect(wrapper.emitted("addProvider")).toHaveLength(1)
    expect(wrapper.emitted("saveProviderChain")).toHaveLength(1)
    expect(wrapper.emitted("runTriggerScrape")).toHaveLength(1)
  })

  it("renders provider list fallback and saving/error feedback", () => {
    const wrapper = mount(SettingsMetadataSection, {
      props: {
        ...baseProps,
        canPickSpecifiedMetadata: false,
        metadataMovieSaving: true,
        metadataMovieError: "metadata save failed",
      },
    })

    expect(wrapper.text()).toContain("settings.metadataMovieProviderNoList")
    expect(wrapper.text()).toContain("settings.metadataMovieProviderSyncing")
    expect(wrapper.text()).toContain("metadata save failed")
    expect(wrapper.find("[data-provider-select]").exists()).toBe(false)
  })
})
