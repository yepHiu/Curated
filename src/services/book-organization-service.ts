import { httpClient } from "@/api/http-client"
import type { BookOrganizationItem, BookOrganizationService } from "./contracts/book-organization-service"
import { mockBookOrganizationService } from "./adapters/mock/mock-book-organization-service"

const path = (id: string) => `/ai/book-organizations/${encodeURIComponent(id)}`
export const webBookOrganizationService: BookOrganizationService = {
  start: request => httpClient.post("/ai/book-organizations", request),
  async items(id, offset = 0) { return (await httpClient.get<{ items: BookOrganizationItem[] }>(`${path(id)}/items`, { offset, limit: 25 })).items },
  control: (id, operation) => httpClient.post(`${path(id)}/${operation}`),
  review: (id, review) => httpClient.post(`${path(id)}/review`, review),
}
export const useBookOrganizationService = () => import.meta.env.VITE_USE_WEB_API === "true" ? webBookOrganizationService : mockBookOrganizationService
