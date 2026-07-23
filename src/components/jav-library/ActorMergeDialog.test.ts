import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { ActorMergePreviewDTO } from "@/api/types"
import ActorMergeDialog from "./ActorMergeDialog.vue"

const serviceMocks = vi.hoisted(() => ({
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

    await wrapper.get("#actor-merge-target").setValue("Target")
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
    await wrapper.get("#actor-merge-target").setValue("Target")
    await buttonContaining(wrapper, "actors.merge.previewAction").trigger("click")
    await flushPromises()

    expect(wrapper.text()).toContain("too many links")
    expect(buttonContaining(wrapper, "actors.merge.confirmAction").attributes()).toHaveProperty(
      "disabled",
    )
  })
})
