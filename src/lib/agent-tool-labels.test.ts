import { describe, expect, it } from "vitest"
import { agentToolI18nKey, isAgentProcessTool, uniqueProcessToolNames } from "./agent-tool-labels"

describe("agent tool labels", () => {
  it("hides present_movies from the process list", () => {
    expect(isAgentProcessTool("present_movies")).toBe(false)
    expect(isAgentProcessTool("search_movies")).toBe(true)
    expect(uniqueProcessToolNames(["search_movies", "present_movies", "search_movies"])).toEqual([
      "search_movies",
    ])
  })

  it("maps known tools to product copy keys", () => {
    expect(agentToolI18nKey("search_movies")).toBe("agentWindow.tools.searchMovies")
    expect(agentToolI18nKey("unknown_tool")).toBe("agentWindow.tools.generic")
    expect(agentToolI18nKey("save_movie_comment")).toBe("agentWindow.tools.saveComment")
    expect(agentToolI18nKey("create_saved_view")).toBe("agentWindow.tools.createSavedView")
    expect(agentToolI18nKey("search_provider_titles")).toBe("agentWindow.tools.searchProviderTitles")
    expect(agentToolI18nKey("get_source_page")).toBe("agentWindow.tools.getSourcePage")
  })
})
