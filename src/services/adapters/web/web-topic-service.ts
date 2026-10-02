import { i18n } from "@/i18n"
import { httpClient } from "@/api/http-client"
import type { MovieListItemDTO } from "@/api/types"
import type { TagOrganizationStats, LibraryTopic, TagOrganizationItem, TagOrganizationJob, TopicLibraryService, TagOrganizationService } from "@/services/contracts/topic-service"
import { mapMovieListItem } from "./mappers"

export const webTopicLibrary: TopicLibraryService = {
  /** 分页读取目录，包含隐藏主题以供恢复。 */
  async listTopics() {
    const topics: LibraryTopic[] = []
    for (let offset = 0; ; offset += 100) {
      const page = await httpClient.get<{ items: LibraryTopic[] }>(`/library/topics?limit=100&offset=${offset}`)
      topics.push(...page.items)
      if (page.items.length < 100) return topics
    }
  },
  /** 首页只读取持久题材和代表影片。 */
  async getHomepageTopics() {
    const result = await httpClient.get<{ items: { topic: LibraryTopic; movies: MovieListItemDTO[] }[] }>("/homepage/topics")
    return result.items.map((group) => {
      // 沿用既有媒体 URL 和偏好映射。
      return { topic: group.topic, movies: group.movies.map(mapMovieListItem) }
    })
  },
  /** 按稳定 ID 读取题材，不回退到原始标签筛选。 */
  getTopic(id) { return httpClient.get<LibraryTopic>(`/library/topics/${encodeURIComponent(id)}`) },
  /** 隐藏只影响展示。 */
  async setTopicHidden(id, hidden) { await httpClient.patch(`/library/topics/${encodeURIComponent(id)}`, { hidden }) },
}

export const webTagOrganization: TagOrganizationService = {
  /** Read problems awaiting an explicit retry. */
  async getTagOrganizationIssues(offset = 0) { return (await httpClient.get<{ items: TagOrganizationItem[] }>(`/ai/tag-organizations/issues?limit=25&offset=${offset}`)).items },
  /** Read actual library coverage without starting AI work. */
  getTagOrganizationStats() { return httpClient.get<TagOrganizationStats>("/ai/tag-organizations/stats") },
  /** 分页读取可复核的归类依据。 */
  async getTagOrganizationItems(id, offset = 0) { return (await httpClient.get<{ items: TagOrganizationItem[] }>(`/ai/tag-organizations/${encodeURIComponent(id)}/items?limit=25&offset=${offset}`)).items },
  /** 重连读取快照，不重放完成提示。 */
  async listTagOrganizations() { return (await httpClient.get<{ items: TagOrganizationJob[] }>("/ai/tag-organizations")).items },
  /** 显式发起后后台独立运行，按钮忙碌态防止重复提交。 */
  startTagOrganization(scope, movieIds) { return httpClient.post<TagOrganizationJob>("/ai/tag-organizations", { scope, movieIds, locale: i18n.global.locale.value, requestId: crypto.randomUUID() }) },
  /** 取消不撤销已应用结果。 */
  cancelTagOrganization(id) { return httpClient.post<TagOrganizationJob>(`/ai/tag-organizations/${encodeURIComponent(id)}/cancel`) },
  /** 重试只处理后端保留的未完成项。 */
  retryTagOrganization(id) { return httpClient.post<TagOrganizationJob>(`/ai/tag-organizations/${encodeURIComponent(id)}/retry`) },
  /** 恢复未被后续人工编辑覆盖的用户标签。 */
  undoTagOrganization(id) { return httpClient.post<{ restored: number; conflicts: number }>(`/ai/tag-organizations/${encodeURIComponent(id)}/undo`) },
}
