import { describe, expect, it } from "vitest"
import actorProfileCardSource from "../components/jav-library/ActorProfileCard.vue?raw"
import curatedFramesLibrarySource from "../components/jav-library/CuratedFramesLibrary.vue?raw"
import movieCommentSectionSource from "../components/jav-library/MovieCommentSection.vue?raw"
import playerPageSource from "../components/jav-library/PlayerPage.vue?raw"
import settingsPageSource from "../components/jav-library/SettingsPage.vue?raw"
import settingsHomepageDevToolsSource from "../components/jav-library/settings/SettingsHomepageDevTools.vue?raw"

const componentSources = [
  { file: "src/components/jav-library/ActorProfileCard.vue", source: actorProfileCardSource },
  { file: "src/components/jav-library/MovieCommentSection.vue", source: movieCommentSectionSource },
  { file: "src/components/jav-library/PlayerPage.vue", source: playerPageSource },
  { file: "src/components/jav-library/CuratedFramesLibrary.vue", source: curatedFramesLibrarySource },
  { file: "src/components/jav-library/SettingsPage.vue", source: settingsPageSource },
  {
    file: "src/components/jav-library/settings/SettingsHomepageDevTools.vue",
    source: settingsHomepageDevToolsSource,
  },
]

describe("library service boundaries", () => {
  it("keeps library UI components behind the LibraryService contract", () => {
    const offenders = componentSources
      .filter(({ source }) => source.includes("@/api/endpoints"))
      .map(({ file }) => file)

    expect(offenders).toEqual([])
  })
})
