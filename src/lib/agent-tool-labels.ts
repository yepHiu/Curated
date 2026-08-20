/** Tools shown as a user-facing library lookup. present_movies is a product card, not a process step. */
export const AGENT_PROCESS_HIDDEN_TOOLS = new Set(["present_movies"])

export const AGENT_TOOL_I18N_KEYS: Record<string, string> = {
  get_library_overview: "agentWindow.tools.libraryOverview",
  search_movies: "agentWindow.tools.searchMovies",
  get_movie_detail: "agentWindow.tools.movieDetail",
  list_actors: "agentWindow.tools.listActors",
  get_actor_profile: "agentWindow.tools.actorProfile",
  get_insights_overview: "agentWindow.tools.insightsOverview",
  get_insights_breakdown: "agentWindow.tools.insightsBreakdown",
  get_watch_history: "agentWindow.tools.watchHistory",
  search_curated_frames: "agentWindow.tools.searchFrames",
  get_curated_frames_stats: "agentWindow.tools.frameStats",
  get_task_status: "agentWindow.tools.taskStatus",
  save_movie_comment: "agentWindow.tools.saveComment",
}

export function isAgentProcessTool(name: string): boolean {
  return Boolean(name) && !AGENT_PROCESS_HIDDEN_TOOLS.has(name)
}

export function agentToolI18nKey(name: string): string {
  return AGENT_TOOL_I18N_KEYS[name] || "agentWindow.tools.generic"
}

export function uniqueProcessToolNames(names: string[]): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const name of names) {
    if (!isAgentProcessTool(name) || seen.has(name)) continue
    seen.add(name)
    out.push(name)
  }
  return out
}
