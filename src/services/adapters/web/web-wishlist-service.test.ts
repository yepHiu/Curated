import { describe, expect, it, vi } from "vitest"
import { webWishlistService } from "./web-wishlist-service"
import { httpClient } from "@/api/http-client"
vi.mock("@/api/http-client", () => { /* 隔离联网验证 HTTP 契约。 */ return { httpClient: { get: vi.fn(), patch: vi.fn(), post: vi.fn(), delete: vi.fn() }, resolveApiBaseUrl: () => { /* 使用测试后端。 */ return "http://localhost:8080/api" } } })
describe("wishlist service", () => { /* 独立愿望接口不伪装影片操作。 */
  it("routes patches through wishlist and preserves version", async () => { /* 防止并发用户修改无条件覆盖。 */ await webWishlistService.patch("a/b", { version: 3, completed: true }); expect(httpClient.patch).toHaveBeenCalledWith("/wishlist/items/a%2Fb", { version: 3, completed: true }) })
  it("rejects remote image destinations", () => { /* 页面图片必须来自授权资产端点。 */ expect(webWishlistService.assetUrl("https://example.com/image")).toBe(""); expect(webWishlistService.assetUrl("/api/wishlist/items/a/assets/b")).toBe("http://localhost:8080/api/wishlist/items/a/assets/b") })
})
