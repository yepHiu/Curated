import { afterEach, expect, it, vi } from "vitest"

afterEach(() => {
  localStorage.removeItem("curated-mock-photo-tags")
  localStorage.removeItem("curated-mock-photo-ratings-v1")
  localStorage.removeItem("curated-mock-photo-titles-v1")
  vi.resetModules()
})

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

it('persists photo comments across service reloads', async () => {
  localStorage.removeItem('curated-mock-photo-comments-v1')
  const { mockPhotoLibraryService: service } = await import('./mock-photo-library-service')
  const saved = await service.putPhotoComment('mock-photo-1', { body: '  photo note  ' })
  expect(saved.body).toBe('photo note')
  vi.resetModules()
  const { mockPhotoLibraryService: reloaded } = await import('./mock-photo-library-service')
  await expect(reloaded.getPhotoComment('mock-photo-1')).resolves.toMatchObject({ body: 'photo note' })
})

it('persists photo ratings across service reloads', async () => {
  localStorage.removeItem('curated-mock-photo-ratings-v1')
  const { mockPhotoLibraryService: service } = await import('./mock-photo-library-service')
  const saved = await service.patchPhoto('mock-photo-1', { rating: 3.5 })
  expect(saved.rating).toBe(3.5)
  vi.resetModules()
  const { mockPhotoLibraryService: reloaded } = await import('./mock-photo-library-service')
  expect((await reloaded.loadPhotoDetail('mock-photo-1'))!.rating).toBe(3.5)
  await reloaded.patchPhoto('mock-photo-1', { rating: null })
  vi.resetModules()
  const { mockPhotoLibraryService: cleared } = await import('./mock-photo-library-service')
  expect((await cleared.loadPhotoDetail('mock-photo-1'))!.rating).toBeNull()
})

it('persists photo display titles across service reloads', async () => {
  localStorage.removeItem('curated-mock-photo-titles-v1')
  const { mockPhotoLibraryService: service } = await import('./mock-photo-library-service')
  const saved = await service.patchPhoto('mock-photo-1', { title: ' 展示写真 ' })
  expect(saved.title).toBe('展示写真')
  vi.resetModules()
  const { mockPhotoLibraryService: reloaded } = await import('./mock-photo-library-service')
  expect((await reloaded.loadPhotoDetail('mock-photo-1'))!.title).toBe('展示写真')
})
