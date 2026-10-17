import { mount } from "@vue/test-utils"
import { expect, it, vi } from "vitest"
import BookMetadataFacts from "./BookMetadataFacts.vue"

vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))

it("keeps review evidence non-interactive until explicitly used for library browsing", async () => {
  const wrapper = mount(BookMetadataFacts, { props: {
    evidence: true,
    metadata: { cleanTitle: "作品", translatedTitle: "作品", authors: [{ name: "青井", evidence: "作者 青井" }] },
  } })
  expect(wrapper.text()).toContain("作者 青井")
  expect(wrapper.find("button").exists()).toBe(false)
  await wrapper.setProps({ browseable: true, evidence: false })
  await wrapper.get("button").trigger("click")
  expect(wrapper.emitted("browseByTag")).toEqual([["青井"]])
  wrapper.unmount()
})
