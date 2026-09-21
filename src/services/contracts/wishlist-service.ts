import type { WishlistItem, WishlistPage, WishlistPatch, WishlistQuery, WishlistToken } from "@/domain/wishlist/types"
export interface WishlistServiceContract {
  list(query?: WishlistQuery): Promise<WishlistPage>
  get(id: string): Promise<WishlistItem>
  patch(id: string, patch: WishlistPatch): Promise<WishlistItem>
  remove(id: string): Promise<void>
  refresh(id: string): Promise<void>
  tokens(): Promise<WishlistToken[]>
  createToken(name: string): Promise<WishlistToken>
  revokeToken(id: string): Promise<void>
  assetUrl(path: string): string
  readonly integrationsAvailable: boolean
}
