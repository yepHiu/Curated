import type { WishlistItem, WishlistPage, WishlistPatch, WishlistPlaybackResponse, WishlistQuery } from "@/domain/wishlist/types"
export interface WishlistServiceContract {
  list(query?: WishlistQuery): Promise<WishlistPage>
  get(id: string): Promise<WishlistItem>
  patch(id: string, patch: WishlistPatch): Promise<WishlistItem>
  remove(id: string): Promise<void>
  refresh(id: string): Promise<void>
  playback(id: string): Promise<WishlistPlaybackResponse>
  assetUrl(path: string): string
  readonly integrationsAvailable: boolean
}
