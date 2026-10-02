import type { Movie } from "@/domain/movie/types"
import type { HomepageTopicGroup, LibraryTopic, TagOrganizationJob, TagOrganizationItem, TagOrganizationService, TopicLibraryService } from "@/services/contracts/topic-service"

interface MockTopicBatch { before: Record<string, string[]>; after: Record<string, string[]>; items: TagOrganizationItem[] }
interface MockTopicAnalysis { jobId: string; fingerprint: string; unresolved: boolean }
interface MockTopicState { issues: Record<string, TagOrganizationItem>; analysis: Record<string, MockTopicAnalysis>; jobs: TagOrganizationJob[]; topics: LibraryTopic[]; batches: Record<string, MockTopicBatch> }
const key = "curated-mock-topic-organization-v2"

/** Mock 状态和真实库分离，刷新后仍可查看结果。 */
function loadState(): MockTopicState {
  try { const value = JSON.parse(localStorage.getItem(key) ?? "null"); if (value?.jobs && value?.topics && value?.batches) return { ...value, analysis: value.analysis ?? {}, issues: value.issues ?? {} } } catch { /* 不完整旧数据降级为空。 */ }
  return { jobs: [], topics: [], batches: {}, analysis: {}, issues: {} }
}
const state = loadState()
/** 持久化模拟任务，不调用模型。 */
function saveState() { localStorage.setItem(key, JSON.stringify(state)) }

/** Source-only fingerprint; user preference edits do not trigger reorganization. */
function sourceFingerprint(movie: Movie) {
  return JSON.stringify([movie.displayScrapeFallback?.title ?? movie.title, movie.displayScrapeFallback?.summary ?? movie.summary ?? "", [...movie.tags].sort()])
}

function analysisStatus(movie: Movie) {
  if (state.issues[movie.id]) return "issues"
  const analysis = state.analysis[movie.id]
  return !analysis ? "unorganized" : analysis.fingerprint === sourceFingerprint(movie) ? "organized" : "outdated"
}

/** 提供确定性模拟，所有写入均经既有用户标签接口。 */
export function createMockTopicServices(movies: () => readonly Movie[], patch: (id: string, tags: string[]) => void | Promise<unknown>): TopicLibraryService & TagOrganizationService {
  return {
    /** Persisted problem queue is not inferred from the current list page. */
    async getTagOrganizationIssues(offset = 0) {
      return movies().filter((movie) => !movie.trashedAt && state.issues[movie.id]).sort((a,b) => a.id.localeCompare(b.id)).slice(offset,offset+25).map((movie) => ({ ...state.issues[movie.id]!, title: movie.title }))
    },
    /** Partition active movies independently of whether a tag was added. */
    async getTagOrganizationStats() {
      const stats = { total: 0, organized: 0, unorganized: 0, outdated: 0, unresolved: 0, needsAttention: 0 }
      for (const movie of movies()) {
        if (movie.trashedAt) continue
        const status = analysisStatus(movie)
        stats.total++; stats[status === "issues" ? "needsAttention" : status]++
        if (status === "organized" && state.analysis[movie.id]?.unresolved) stats.unresolved++
      }
      return stats
    },
    /** 主题目录保留隐藏条目。 */
    async listTopics() { return structuredClone(state.topics) },
    /** 结果页只显示模拟证据，分页与 Web 一致。 */
    async getTagOrganizationItems(id, offset = 0) { return structuredClone(state.batches[id]?.items.slice(offset, offset + 25) ?? []) },
    /** 从已整理的用户标签生成代表影片。 */
    async getHomepageTopics() {
      const seen = new Set<string>()
      const groups: HomepageTopicGroup[] = []
      for (const topic of state.topics) {
        if (topic.hidden) continue
        const members = movies().filter((m) => { /* 仅 userTags 能满足主题。 */ return m.userTags.includes(topic.name) })
        topic.movieCount = members.length
        const representatives = members.filter((m) => { /* 同屏去重不改真实成员。 */ return !seen.has(m.id) }).slice(0, 6)
        if (representatives.length < 2) continue
        representatives.forEach((m) => { /* 标记已展示影片。 */ seen.add(m.id) })
        groups.push({ topic: { ...topic }, movies: representatives })
        if (groups.length === 3) break
      }
      return groups
    },
    /** 稳定模拟 ID 定位主题。 */
    async getTopic(id) { const topic = state.topics.find((t) => { /* 精确匹配。 */ return t.id === id }); if (!topic) throw new Error("Topic not found"); return { ...topic, movieCount: movies().filter((m) => { /* 动态成员计数。 */ return m.userTags.includes(topic.name) }).length } },
    /** 隐藏不改标签。 */
    async setTopicHidden(id, hidden) { const topic = state.topics.find((t) => { /* 精确匹配。 */ return t.id === id }); if (topic) topic.hidden = hidden; saveState() },
    /** 返回副本避免界面误改持久状态。 */
    async listTagOrganizations() { return state.jobs.map((j) => { /* 拷贝快照。 */ return { ...j } }) },
    /** 模拟任务用源标签作为确定性题材样本，写入仅 userTags。 */
    async startTagOrganization(scope, movieIds) {
      const selected = movies().filter((m) => { /* 模拟范围快照。 */ return !m.trashedAt && (scope === "all" || (scope === "selected" ? movieIds?.includes(m.id) : analysisStatus(m) === scope)) })
      if (!selected.length) throw new Error("AI_ORGANIZATION_NO_MOVIES")
      const now = new Date().toISOString()
      const job: TagOrganizationJob = { id: crypto.randomUUID(), taskId: "", status: "running", stage: "classifying", triggerReason: "manual", total: selected.length, processed: 0, succeeded: 0, unresolved: 0, failed: 0, revision: 1, createdAt: now, updatedAt: now }
      job.taskId = job.id; state.jobs.unshift(job)
      const batch: MockTopicBatch = { before: {}, after: {}, items: [] }; state.batches[job.id] = batch
      for (const movie of selected) {
        if ((movie.title.length + (movie.summary?.length ?? 0)) > 24000) {
          const item: TagOrganizationItem = { movieId: movie.id, title: movie.title, status: "failed", reason: "SOURCE_TOO_LONG", evidence: [] }
          state.issues[movie.id] = item; batch.items.push(item); job.failed++; job.processed++; continue
        }
        const names = movie.tags.slice(0, 3)
        const before = [...movie.userTags]
        const fingerprint = sourceFingerprint(movie)
        const after = [...new Set([...movie.userTags, ...names])]
        if (JSON.stringify([...before].sort()) !== JSON.stringify([...after].sort())) {
          batch.before[movie.id] = before
          await patch(movie.id, after); batch.after[movie.id] = after
        }
        delete state.issues[movie.id]
        state.analysis[movie.id] = { jobId: job.id, fingerprint, unresolved: names.length === 0 }
        for (const name of names) if (!state.topics.some((t) => { /* 复用规范名称。 */ return t.name === name })) state.topics.push({ id: `mock-topic-${encodeURIComponent(name)}`, name, description: "", movieCount: 0, hidden: false })
        batch.items.push({ movieId: movie.id, title: movie.title, status: names.length ? "succeeded" : "unresolved", reason: names.length ? "" : "INSUFFICIENT_EVIDENCE", evidence: names.map((topic) => { /* 只引用实际源标签。 */ return { topic, field: "metadataTags", quote: topic } }) })
        job.processed++; if (names.length) job.succeeded++; else job.unresolved++
      }
      job.status = job.failed ? "partial_failed" : "completed"; job.stage = "finished"; job.revision++; saveState(); return { ...job }
    },
    /** 模拟取消保留已处理结果。 */
    async cancelTagOrganization(id) { const job = state.jobs.find((j) => { /* 精确匹配。 */ return j.id === id }); if (!job) throw new Error("Task not found"); if (["running", "queued"].includes(job.status)) job.status = "cancelled"; saveState(); return { ...job } },
    /** 模拟已完成任务无需重复执行。 */
    async retryTagOrganization(id) { const job = state.jobs.find((j) => { /* 精确匹配。 */ return j.id === id }); if (!job) throw new Error("Task not found"); return { ...job } },
    /** 并发人工修改优先于模拟撤销。 */
    async undoTagOrganization(id) {
      const batch = state.batches[id]
      if (!batch) throw new Error("Task not found")
      let restored = 0, conflicts = 0
      for (const movie of movies()) {
        const before = batch.before[movie.id]; if (!before) continue
        if (JSON.stringify(movie.userTags) !== JSON.stringify(batch.after[movie.id])) { conflicts++; continue }
        await patch(movie.id, before); delete batch.before[movie.id]; restored++
        if (state.analysis[movie.id]?.jobId === id) delete state.analysis[movie.id]
      }
      saveState(); return { restored, conflicts }
    },
  }
}
