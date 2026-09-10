import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => {
      const messages: Record<string, string> = {
        "settings.moreActions": "More actions",
        "settings.comicLibraryPathScan": "Scan comics",
        "settings.comicLibraryPathRemove": "Remove",
      }
      return messages[key] ?? key
    },
  }),
}))

vi.mock("lucide-vue-next", () => ({
  MoreVertical: { name: "MoreVertical", template: "<span />" },
  RefreshCw: { name: "RefreshCw", template: "<span />" },
  Trash2: { name: "Trash2", template: "<span />" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    props: ["disabled", "ariaLabel"],
    emits: ["click"],
    template:
      "<button v-bind='$attrs' :disabled='disabled' :aria-label='ariaLabel' @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: { name: "DropdownMenu", template: "<div><slot /></div>" },
  DropdownMenuContent: { name: "DropdownMenuContent", template: "<div><slot /></div>" },
  DropdownMenuGroup: { name: "DropdownMenuGroup", template: "<div><slot /></div>" },
  DropdownMenuTrigger: { name: "DropdownMenuTrigger", template: "<div><slot /></div>" },
  DropdownMenuItem: {
    name: "DropdownMenuItem",
    props: ["disabled", "variant"],
    emits: ["click"],
    template:
      "<button v-bind='$attrs' :disabled='disabled' :data-variant='variant' @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

async function mountComponent(props?: Record<string, unknown>) {
  const mod = await import("./SettingsComicLibraryPathActions.vue")
  return mount(mod.default, {
    props: {
      path: {
        id: "comic-path-a",
        title: "comic",
        path: "D:/Comics",
      },
      ...props,
    },
  })
}

describe("SettingsComicLibraryPathActions", () => {
  it("wraps comic path actions in a more-actions menu", async () => {
    const wrapper = await mountComponent()

    expect(wrapper.find('button[aria-label="More actions"]').exists()).toBe(true)
    expect(wrapper.text()).toContain("Scan comics")
    expect(wrapper.text()).toContain("Remove")
  })

  it("emits scan and remove actions for the selected comic path", async () => {
    const wrapper = await mountComponent()

    await wrapper.get("[data-scan-comic-path='comic-path-a']").trigger("click")
    await wrapper.get("[data-remove-comic-path='comic-path-a']").trigger("click")

    expect(wrapper.emitted("scan")).toEqual([
      [
        {
          id: "comic-path-a",
          title: "comic",
          path: "D:/Comics",
        },
      ],
    ])
    expect(wrapper.emitted("remove")).toEqual([["comic-path-a"]])
  })

  it("disables scan while that comic path is scanning", async () => {
    const wrapper = await mountComponent({ scanBusy: true })

    expect(wrapper.get("[data-scan-comic-path='comic-path-a']").attributes("disabled")).toBeDefined()
  })
})
