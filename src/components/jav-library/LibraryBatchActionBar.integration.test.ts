import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import LibraryBatchActionBar from "./LibraryBatchActionBar.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe("LibraryBatchActionBar integration", () => {
  it("submits the dialog tag when clicking the append action button", async () => {
    const wrapper = mount(LibraryBatchActionBar, {
      attachTo: document.body,
      props: {
        mode: "library",
        selectedCount: 2,
        useWebApi: true,
        scrapeProgress: null,
        scrapeBusy: false,
        operationBusy: false,
      },
    })

    const openDialogButton = wrapper
      .findAll("button")
      .find((button) => button.text().includes("library.batchAddTag"))

    expect(openDialogButton).toBeDefined()

    await openDialogButton!.trigger("click")

    const input = document.body.querySelector(
      'input[placeholder="library.batchTagPlaceholder"]',
    ) as HTMLInputElement | null
    expect(input).not.toBeNull()

    input!.value = "batch-tag"
    input!.dispatchEvent(new Event("input", { bubbles: true }))
    await wrapper.vm.$nextTick()

    const submitButton = [...document.body.querySelectorAll("button")].find((button) =>
      button.textContent?.includes("library.batchTagSubmit"),
    ) as HTMLButtonElement | undefined

    expect(submitButton).toBeDefined()

    submitButton!.click()
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted("addUserTag")).toEqual([["batch-tag"]])

    wrapper.unmount()
  })
})
