import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) => {
      const messages: Record<string, string> = {
        "settings.moreActions": "More actions",
        "settings.openPath": "Open folder",
        "settings.editTitle": "Rename",
        "settings.rescan": "Rescan",
        "settings.removePathConfirmAction": "Remove",
        "settings.openPathAria": `Open ${params?.title ?? ""} in file manager`,
      }
      return messages[key] ?? key
    },
  }),
}))

vi.mock("lucide-vue-next", () => ({
  FolderOpen: { name: "FolderOpen", template: "<span />" },
  MoreVertical: { name: "MoreVertical", template: "<span />" },
  Pencil: { name: "Pencil", template: "<span />" },
  RefreshCw: { name: "RefreshCw", template: "<span />" },
  Trash2: { name: "Trash2", template: "<span />" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    props: ["disabled", "ariaLabel"],
    emits: ["click"],
    template:
      "<button :disabled=\"disabled\" :aria-label=\"ariaLabel\" @click=\"$emit('click', $event)\"><slot /></button>",
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
      "<button :disabled=\"disabled\" :data-variant=\"variant\" @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

async function mountComponent(props?: Record<string, unknown>) {
  const mod = await import("./SettingsLibraryPathActions.vue")
  return mount(mod.default, {
    props: {
      path: {
        id: "library-a",
        title: "Primary archive",
        path: "D:/Media/JAV/Main",
      },
      ...props,
    },
  })
}

describe("SettingsLibraryPathActions", () => {
  it("renders a more-actions trigger and grouped menu items", async () => {
    const wrapper = await mountComponent()

    expect(wrapper.find('button[aria-label="More actions"]').exists()).toBe(true)
    expect(wrapper.text()).toContain("Open folder")
    expect(wrapper.text()).toContain("Rename")
    expect(wrapper.text()).toContain("Rescan")
    expect(wrapper.text()).toContain("Remove")
  })

  it("disables reveal and rescan menu items when those actions are busy", async () => {
    const wrapper = await mountComponent({
      revealBusy: true,
      scanBusy: true,
    })

    const buttons = wrapper.findAll("button")
    const openFolder = buttons.find((button) => button.text().includes("Open folder"))
    const rescan = buttons.find((button) => button.text().includes("Rescan"))

    expect(openFolder?.attributes("disabled")).toBeDefined()
    expect(rescan?.attributes("disabled")).toBeDefined()
  })
})
