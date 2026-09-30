import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { ActorMergePreviewDTO } from "@/api/types"
import ActorMergeDialog from "./ActorMergeDialog.vue"
import { HttpClientError } from "@/api/http-client"

const serviceMocks = vi.hoisted(() => ({
  listActors: vi.fn(),
  getActorProfile: vi.fn(),
  previewActorMerge: vi.fn(),
  applyActorMerge: vi.fn(),
}))
const toastMock = vi.hoisted(() => vi.fn())

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => serviceMocks,
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: toastMock,
}))

const SlotStub = { template: "<div><slot /></div>" }
const ButtonStub = {
  inheritAttrs: false,
  props: ["disabled"],
  emits: ["click"],
  template:
    '<button v-bind="$attrs" :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>',
}
const InputStub = {
  inheritAttrs: false,
  props: ["modelValue", "disabled"],
  emits: ["update:modelValue", "keydown"],
  template:
    '<input v-bind="$attrs" :value="modelValue" :disabled="disabled" @input="$emit(\'update:modelValue\', $event.target.value)" @keydown="$emit(\'keydown\', $event)" />',
}

function preview(): ActorMergePreviewDTO {
  return {
    previewToken: "preview-token",
    source: { id: 1, name: "Source", aliases: ["Old Source"] },
    target: { id: 2, name: "Target", aliases: [] },
    movies: { sourceCount: 2, targetCount: 2, duplicateCount: 1, resultCount: 3 },
    userTags: { source: ["source-tag"], target: [], result: ["source-tag"] },
    externalLinks: { source: [], target: [], result: [] },
    recommendationFeedback: {
      sourceCount: 1,
      targetCount: 0,
      duplicateCount: 0,
      resultCount: 1,
    },
    curatedFramesAffected: 1,
    aliasesToMove: ["Source", "Old Source"],
    profileFields: [
      {
        field: "summary",
        sourceValue: "source summary",
        targetValue: "target summary",
        defaultSelection: "target",
        conflict: true,
      },
    ],
    canApply: true,
    blockingReasons: [],
    requiredDecisions: ["summary"],
  }
}

function mountDialog() {
  return mount(ActorMergeDialog, {
      props: { open: true, sourceName: "Source" },
      global: {
        stubs: {
          Badge: SlotStub,
          Button: ButtonStub,
          Dialog: SlotStub,
          DialogContent: SlotStub,
          DialogDescription: SlotStub,
          DialogFooter: SlotStub,
          DialogHeader: SlotStub,
          DialogTitle: SlotStub,
          Input: InputStub,
          GitMerge: true,
          Loader2: true,
          RefreshCw: true,
        },
      },
  })
}

function buttonContaining(wrapper: ReturnType<typeof mountDialog>, text: string) {
  const button = wrapper.findAll("button").find((candidate) => candidate.text().includes(text))
  if (!button) throw new Error(`button not found: ${text}`)
  return button
}

describe("ActorMergeDialog", () => {
  beforeEach(() => {
    serviceMocks.listActors.mockReset().mockResolvedValue({ total: 2, actors: [{ name: "Source", movieCount: 2 }, { name: "Target", movieCount: 2 }] })
    serviceMocks.getActorProfile.mockReset().mockResolvedValue({ name: "Source" })
    serviceMocks.previewActorMerge.mockReset()
    serviceMocks.applyActorMerge.mockReset()
    toastMock.mockReset()
  })

  it("requires a read-only preview and explicit conflict choice before applying", async () => {
    serviceMocks.previewActorMerge.mockResolvedValue(preview())
    serviceMocks.applyActorMerge.mockResolvedValue({
      id: "amrg_test",
      sourceName: "Source",
      targetName: "Target",
    })
    const wrapper = mountDialog()

    await flushPromises()
    await buttonContaining(wrapper, "Target").trigger("click")
    await buttonContaining(wrapper, "actors.merge.previewAction").trigger("click")
    await flushPromises()

    expect(serviceMocks.previewActorMerge).toHaveBeenCalledWith({
      sourceName: "Source",
      targetName: "Target",
    })
    expect(wrapper.text()).toContain("actors.merge.conflictsTitle")
    expect(buttonContaining(wrapper, "actors.merge.confirmAction").attributes()).toHaveProperty(
      "disabled",
    )

    await wrapper.get('input[type="radio"][value="source"]').setValue()
    await buttonContaining(wrapper, "actors.merge.confirmAction").trigger("click")
    await flushPromises()

    expect(serviceMocks.applyActorMerge).toHaveBeenCalledWith({
      sourceName: "Source",
      targetName: "Target",
      previewToken: "preview-token",
      confirm: true,
      profileDecisions: { summary: "source" },
    })
    expect(wrapper.emitted("merged")?.[0]).toEqual(["Target"])
    expect(wrapper.emitted("update:open")?.at(-1)).toEqual([false])
    expect(toastMock).toHaveBeenCalled()
  })

  it("renders blocking reasons and never enables apply", async () => {
    serviceMocks.previewActorMerge.mockResolvedValue({
      ...preview(),
      canApply: false,
      blockingReasons: [{ code: "ACTOR_MERGE_LINK_LIMIT", message: "too many links" }],
      requiredDecisions: [],
      profileFields: [],
    })
    const wrapper = mountDialog()
    await flushPromises()
    await buttonContaining(wrapper, "Target").trigger("click")
    await buttonContaining(wrapper, "actors.merge.previewAction").trigger("click")
    await flushPromises()

    expect(wrapper.text()).toContain("actors.merge.linkLimit")
    expect(buttonContaining(wrapper, "actors.merge.confirmAction").attributes()).toHaveProperty(
      "disabled",
    )
  })

  async function selectAndPreview(wrapper: ReturnType<typeof mountDialog>) {
    await flushPromises()
    await buttonContaining(wrapper, "Target").trigger("click")
    await buttonContaining(wrapper, "actors.merge.previewAction").trigger("click")
    await flushPromises()
  }

  it("reverses the merge direction and clears previous decisions", async () => {
    serviceMocks.previewActorMerge.mockResolvedValue(preview())
    const wrapper = mountDialog()
    await selectAndPreview(wrapper)
    await wrapper.get('input[name="actor-merge-summary"][value="source"]').setValue()
    await wrapper.get('input[name="actor-merge-keep"][value="true"]').setValue()
    expect(wrapper.find("[data-actor-merge-result]").exists()).toBe(false)
    expect(buttonContaining(wrapper, "actors.merge.confirmAction").attributes()).toHaveProperty("disabled")
    await buttonContaining(wrapper, "actors.merge.previewAction").trigger("click")
    expect(serviceMocks.previewActorMerge).toHaveBeenLastCalledWith({ sourceName: "Target", targetName: "Source" })
    wrapper.unmount()
  })

  it("shows aliases and filled values, and submits avatar and provider groups together", async () => {
    const next = preview()
    next.profileFields = [
      ...["avatarRemoteUrl", "avatarLocalPath", "provider", "providerActorId"].map((field) => ({ field, sourceValue: `source-${field}`, targetValue: `target-${field}`, defaultSelection: "target" as const, conflict: true })),
      { field: "summary", sourceValue: "Filled biography", targetValue: "", defaultSelection: "source", conflict: false },
    ]
    next.requiredDecisions = ["avatarRemoteUrl", "avatarLocalPath", "provider", "providerActorId"]
    serviceMocks.previewActorMerge.mockResolvedValue(next)
    serviceMocks.applyActorMerge.mockResolvedValue({ sourceName: "Source", targetName: "Target" })
    const wrapper = mountDialog()
    await selectAndPreview(wrapper)
    expect(wrapper.text()).toContain("Old Source")
    expect(wrapper.text()).toContain("Filled biography")
    expect(wrapper.text()).not.toContain("source-avatarLocalPath")
    await wrapper.get('input[name="actor-merge-avatar"][value="source"]').setValue()
    await wrapper.get('input[name="actor-merge-providerIdentity"][value="target"]').setValue()
    await buttonContaining(wrapper, "actors.merge.confirmAction").trigger("click")
    expect(serviceMocks.applyActorMerge).toHaveBeenCalledWith(expect.objectContaining({ profileDecisions: {
      summary: "source", avatarRemoteUrl: "source", avatarLocalPath: "source", provider: "target", providerActorId: "target",
    } }))
    wrapper.unmount()
  })

  it("discards a late preview after closing and reopening", async () => {
    let resolve!: (value: ActorMergePreviewDTO) => void
    serviceMocks.previewActorMerge.mockReturnValue(new Promise<ActorMergePreviewDTO>((done) => { resolve = done }))
    const wrapper = mountDialog()
    await selectAndPreview(wrapper)
    await wrapper.setProps({ open: false })
    await wrapper.setProps({ open: true })
    resolve(preview())
    await flushPromises()
    expect(wrapper.find("[data-actor-merge-result]").exists()).toBe(false)
    expect(wrapper.find("#actor-merge-target").exists()).toBe(true)
    expect(serviceMocks.applyActorMerge).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it("discards a late preview after changing the retained actor", async () => {
    let resolve!: (value: ActorMergePreviewDTO) => void
    serviceMocks.previewActorMerge.mockReturnValue(new Promise<ActorMergePreviewDTO>((done) => { resolve = done }))
    const wrapper = mountDialog()
    await selectAndPreview(wrapper)
    await wrapper.get('input[name="actor-merge-keep"][value="true"]').setValue()
    resolve(preview())
    await flushPromises()
    expect(wrapper.find("[data-actor-merge-result]").exists()).toBe(false)
    expect(buttonContaining(wrapper, "actors.merge.previewAction").attributes()).not.toHaveProperty("disabled")
    wrapper.unmount()
  })

  it("requires a fresh preview after a stale apply and prevents duplicate submissions", async () => {
    serviceMocks.previewActorMerge.mockResolvedValue(preview())
    let reject!: (reason: Error) => void
    serviceMocks.applyActorMerge.mockReturnValue(new Promise((_resolve, fail) => { reject = fail }))
    const wrapper = mountDialog()
    await selectAndPreview(wrapper)
    await wrapper.get('input[name="actor-merge-summary"][value="source"]').setValue()
    const confirm = buttonContaining(wrapper, "actors.merge.confirmAction")
    await confirm.trigger("click")
    await confirm.trigger("click")
    expect(serviceMocks.applyActorMerge).toHaveBeenCalledTimes(1)
    reject(new HttpClientError(409, { code: "ACTOR_MERGE_STALE_PREVIEW", message: "stale", retryable: false }))
    await flushPromises()
    expect(wrapper.text()).toContain("actors.merge.stalePreview")
    expect(wrapper.find("[data-actor-merge-result]").exists()).toBe(false)
    expect(buttonContaining(wrapper, "actors.merge.confirmAction").attributes()).toHaveProperty("disabled")
    wrapper.unmount()
  })

  it("reports a committed merge with failed refresh as a warning and navigates once", async () => {
    serviceMocks.previewActorMerge.mockResolvedValue({ ...preview(), profileFields: [], requiredDecisions: [] })
    serviceMocks.applyActorMerge.mockResolvedValue({ sourceName: "Source", targetName: "Target", refreshFailed: true })
    const wrapper = mountDialog()
    await selectAndPreview(wrapper)
    await buttonContaining(wrapper, "actors.merge.confirmAction").trigger("click")
    await flushPromises()
    expect(toastMock).toHaveBeenCalledWith(expect.stringContaining("actors.merge.refreshFailed"), { variant: "warning" })
    expect(wrapper.emitted("merged")).toEqual([["Target"]])
    expect(wrapper.find("[data-actor-merge-result]").exists()).toBe(false)
    wrapper.unmount()
  })

  it("searches aliases, excludes the current actor, and ignores out-of-order search results", async () => {
    let resolve!: (value: unknown) => void
    serviceMocks.listActors.mockReturnValueOnce(new Promise((done) => { resolve = done }))
    const wrapper = mountDialog()
    await wrapper.get("#actor-merge-target").setValue("Old Target")
    await wrapper.get("#actor-merge-target").trigger("keydown", { key: "Enter" })
    await flushPromises()
    expect(serviceMocks.listActors).toHaveBeenLastCalledWith({ q: "Old Target", sort: "name", limit: 20 })
    resolve({ total: 1, actors: [{ name: "Late actor", movieCount: 1 }] })
    await flushPromises()
    expect(wrapper.text()).not.toContain("Late actor")
    expect(wrapper.findAll("li")).toHaveLength(1)
    await buttonContaining(wrapper, "Target").trigger("click")
    expect(wrapper.text()).toContain("actors.merge.keepActor")
    wrapper.unmount()
  })

  it("recovers from search failure and distinguishes empty results", async () => {
    serviceMocks.listActors.mockRejectedValueOnce(new Error("offline"))
    const wrapper = mountDialog()
    await flushPromises()
    expect(wrapper.text()).toContain("actors.merge.searchFailed")
    serviceMocks.listActors.mockResolvedValueOnce({ total: 0, actors: [] })
    await buttonContaining(wrapper, "actors.merge.retry").trigger("click")
    await flushPromises()
    expect(wrapper.text()).toContain("actors.merge.noResults")
    expect(wrapper.text()).not.toContain("actors.merge.searchFailed")
    wrapper.unmount()
  })
})
