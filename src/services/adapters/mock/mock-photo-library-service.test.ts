import { afterEach, expect, it, vi } from "vitest"

afterEach(() => { localStorage.removeItem('curated-mock-photo-tags'); vi.resetModules() })

it('persists photo tags across service reloads without touching other books', async () => {
  localStorage.removeItem('curated-mock-photo-tags')
  const { mockPhotoLibraryService: service } = await import('./mock-photo-library-service')
  const otherTags = [...service.getPhotoById('mock-photo-2')!.tags]
  const updated = await service.replacePhotoTags('mock-photo-1', [' new ', 'new', '风景'])
  expect(updated.tags).toEqual(['new', '风景'])
  vi.resetModules()
  const { mockPhotoLibraryService: reloaded } = await import('./mock-photo-library-service')
  expect((await reloaded.loadPhotoDetail('mock-photo-1'))!.tags).toEqual(['new', '风景'])
  expect(reloaded.getPhotoById('mock-photo-2')!.tags).toEqual(otherTags)
  await expect(reloaded.replacePhotoTags('mock-photo-1', ['界'.repeat(65)])).rejects.toThrow()
  expect(reloaded.getPhotoById('mock-photo-1')!.tags).toEqual(['new', '风景'])
})
