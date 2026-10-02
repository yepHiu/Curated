import type { Movie } from "@/domain/movie/types"

export interface LibraryTopic {
  id: string
  name: string
  description: string
  movieCount: number
  hidden: boolean
}
export interface HomepageTopicGroup { topic: LibraryTopic; movies: Movie[] }
export interface TagOrganizationJob {
  locale?: "zh-CN" | "en" | "ja"
  id: string
  taskId: string
  status: "queued" | "running" | "blocked" | "completed" | "partial_failed" | "failed" | "cancelled"
  vocabularyProcessed?: number
  vocabularyReady?: boolean
  stage: string
  triggerReason: string
  total: number
  processed: number
  succeeded: number
  unresolved: number
  failed: number
  revision: number
  createdAt: string
  updatedAt: string
  error?: string
}
export interface TopicLibraryService {
  listTopics(): Promise<LibraryTopic[]>
  getHomepageTopics(): Promise<HomepageTopicGroup[]>
  getTopic(id: string): Promise<LibraryTopic>
  setTopicHidden(id: string, hidden: boolean): Promise<void>
}
export interface TagOrganizationItem {
  movieId: string
  title: string
  status: string
  reason: string
  evidence: { topic: string; field: string; quote: string }[]
}
export type TagOrganizationScope = "all" | "selected" | "unorganized" | "outdated" | "issues"
export interface TagOrganizationStats {
  needsAttention: number
  total: number
  organized: number
  unorganized: number
  outdated: number
  /** Successfully analyzed with no matching topic; included in organized. */
  unresolved: number
}
export interface TagOrganizationService {
  getTagOrganizationIssues(offset?: number): Promise<TagOrganizationItem[]>
  getTagOrganizationStats(): Promise<TagOrganizationStats>
  getTagOrganizationItems(id: string, offset?: number): Promise<TagOrganizationItem[]>
  listTagOrganizations(): Promise<TagOrganizationJob[]>
  startTagOrganization(scope: TagOrganizationScope, movieIds?: string[]): Promise<TagOrganizationJob>
  cancelTagOrganization(id: string): Promise<TagOrganizationJob>
  retryTagOrganization(id: string): Promise<TagOrganizationJob>
  undoTagOrganization(id: string): Promise<{ restored: number; conflicts: number }>
}
