import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useCuratedCaptureQueue } from './use-curated-capture-queue'
import type { Movie } from '@/domain/movie/types'

const mocks = vi.hoisted(() => ({ capture: vi.fn(), save: vi.fn(), export: vi.fn() }))
vi.mock('@/lib/curated-frames/save-capture', () => ({ captureCuratedFrameCandidate: mocks.capture, saveCuratedFrameCandidate: mocks.save, exportCuratedFrameCandidate: mocks.export }))
vi.mock('@/lib/curated-frames/db', () => ({ deleteCuratedFrame: vi.fn() }))
vi.mock('@/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }))
afterEach(() => { vi.restoreAllMocks(); vi.clearAllMocks() })

describe('capture queue', () => {
  it('serializes writes, freezes movie data, and retries the same candidate', async () => {
    vi.stubGlobal('URL', { createObjectURL: vi.fn(() => 'blob:test'), revokeObjectURL: vi.fn() })
    let queue!: ReturnType<typeof useCuratedCaptureQueue>
    const wrapper = mount(defineComponent({ setup() { queue = useCuratedCaptureQueue(); return () => null } }))
    const candidate = { id: 'capture-a', blob: new Blob(['a']), positionSec: 12, capturedAt: 'now' }
    mocks.capture.mockResolvedValue({ ok: true, candidate })
    mocks.export.mockResolvedValue(undefined)
    let resolve!: (result: { ok: false; reason: string }) => void
    mocks.save.mockImplementationOnce(() => new Promise(r => { resolve = r })).mockResolvedValue({ ok: true, id: candidate.id, positionSec: 12 })
    const movie = { id: 'movie-a', actors: ['A'] } as Movie
    const video = { videoWidth: 1920, videoHeight: 1080 } as HTMLVideoElement
    const first = queue.prepare(video, movie, 12)!
    movie.id = 'movie-b'; movie.actors.push('B')
    const second = queue.prepare(video, movie, 13)!
    const one = queue.submit(first), two = queue.submit(second)
    await flushPromises()
    expect(mocks.save).toHaveBeenCalledTimes(1)
    expect(mocks.save.mock.calls[0]?.[1]).toMatchObject({ id: 'movie-a', actors: ['A'] })
    resolve({ ok: false, reason: 'offline' })
    await one; await two
    expect(first.phase).toBe('error')
    await queue.submit(first)
    expect(mocks.save.mock.calls[2]?.[0]).toBe(candidate)
    wrapper.unmount(); vi.unstubAllGlobals()
  })

  it('limits allocated pixels before starting another capture', () => {
    mocks.capture.mockImplementation(() => new Promise(() => {}))
    let queue!: ReturnType<typeof useCuratedCaptureQueue>
    const wrapper = mount(defineComponent({ setup() { queue = useCuratedCaptureQueue(); return () => null } }))
    const video = { videoWidth: 3840, videoHeight: 2160 } as HTMLVideoElement
    const movie = { id: 'a', actors: [] } as unknown as Movie
    for (let i = 0; i < 4; i++) expect(queue.prepare(video, movie, i)).toBeDefined()
    expect(queue.prepare(video, movie, 5)).toBeUndefined()
    expect(mocks.capture).toHaveBeenCalledTimes(4)
    wrapper.unmount()
  })
})
