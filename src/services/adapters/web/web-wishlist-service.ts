import { httpClient, resolveApiBaseUrl } from "@/api/http-client"
import type { WishlistServiceContract } from "@/services/contracts/wishlist-service"
const root = "/wishlist/items"
/** 使用统一 HTTP 客户端与 PIN 会话访问愿望能力。 */
export const webWishlistService: WishlistServiceContract = {
  integrationsAvailable: true,
  /** 分页查询条目。 */
  list(query = {}) { return httpClient.get(root, { ...query }) },
  /** 获取独立详情。 */
  get(id) { return httpClient.get(`${root}/${encodeURIComponent(id)}`) },
  /** 按版本写入用户修改。 */
  patch(id, body) { return httpClient.patch(`${root}/${encodeURIComponent(id)}`, body) },
  /** 删除愿望而不删除库影片。 */
  remove(id) { return httpClient.delete(`${root}/${encodeURIComponent(id)}`) },
  /** 合并请求后台重试。 */
  refresh(id) { return httpClient.post(`${root}/${encodeURIComponent(id)}/refresh`) },
  /** 将受保护资产路径映射到当前后端，拒绝外部 URL。 */
  assetUrl(path) { return path.startsWith("/api/wishlist/") ? resolveApiBaseUrl(import.meta.env).replace(/\/api$/, "") + path : "" },
}
