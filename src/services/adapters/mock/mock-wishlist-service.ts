import type { WishlistServiceContract } from "@/services/contracts/wishlist-service"
import type { WishlistItem } from "@/domain/wishlist/types"
const key = "curated-wishlist-mock-v1"
/** Mock 独立存储，不污染本地影片用户偏好。 */
function read(): WishlistItem[] { try { return JSON.parse(localStorage.getItem(key) ?? "[]") as WishlistItem[] } catch { return [] } }
/** 保存 Mock 用户意愿。 */
function write(items: WishlistItem[]) { localStorage.setItem(key, JSON.stringify(items)) }
/** 查找条目，不存在时与真实模式一样失败。 */
function get(id: string) { const item = read().find((value) => { /* 匹配唯一 ID。 */ return value.id === id }); if (!item) throw new Error("Not found"); return item }
/** Mock 只演示管理，不接收真实插件请求。 */
export const mockWishlistService: WishlistServiceContract = {
  integrationsAvailable: false,
  /** 按筛选分页演示愿望列表。 */
  async list(query = {}) { const all = read(); const items = all.filter((item) => { /* 应用与页面一致的状态和搜索。 */ return (!query.status || query.status === "all" || item.status === query.status) && (!query.q || `${item.code} ${item.metadata.title}`.toLowerCase().includes(query.q.toLowerCase())) }); const offset = Number(query.cursor ?? 0); const limit = query.limit ?? 60; return { items: items.slice(offset, offset + limit), total: items.length, pendingCount: all.filter((i) => { /* 只统计待入库意愿。 */ return i.status === "pending" }).length, nextCursor: offset + limit < items.length ? String(offset + limit) : undefined } },
  /** 读取持久条目。 */
  async get(id) { return get(id) },
  /** 验证版本后修改。 */
  async patch(id, patch) { const item = get(id); if (item.version !== patch.version) throw new Error("Version conflict"); Object.assign(item, patch, { version: item.version + 1 }); item.status = item.movieIds.length ? "in_library" : item.completed ? "completed" : "pending"; write(read().map((value) => { /* 替换被编辑的记录。 */ return value.id === id ? item : value })); return item },
  /** 清理指定 Mock 记录。 */
  async remove(id) { write(read().filter((item) => { /* 保留其他记录。 */ return item.id !== id })) },
  /** Mock 不启动网络刮削。 */
  async refresh() { throw new Error("Web API required") },
  /** 只允许应用提供的本地图像。 */
  assetUrl(path) { return path.startsWith("/") && !path.startsWith("//") ? path : "" },
}
