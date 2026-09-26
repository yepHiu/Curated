import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { normalizeServerUrl, probeServer, serverSessionPartition, ServerConnectionStore } from './server-connections'

const dirs: string[] = []
function fixture() {
  const dir = mkdtempSync(path.join(tmpdir(), 'curated-servers-'))
  dirs.push(dir)
  const file = path.join(dir, 'servers.json')
  return { file, store: new ServerConnectionStore(file) }
}
afterEach(() => dirs.splice(0).forEach(dir => rmSync(dir, { recursive: true, force: true })))

describe('Desktop server connections', () => {
  it('adds without overwriting existing records, ids or the last connected server', () => {
    const { store, file } = fixture()
    const first = store.add({ name: 'Home', url: 'nas.local:8081' })
    store.remember(first.id)
    expect(() => store.add({ name: 'Overwrite', url: 'http://nas.local:8081/' })).toThrow()
    expect(() => store.add({ id: first.id, name: 'Overwrite', url: 'other.local' })).toThrow()
    const second = store.add({ name: 'Office', url: 'office.local' })
    expect(new ServerConnectionStore(file).snapshot()).toEqual({ schema: 1, servers: [first, second], lastServerId: first.id })
  })
  it('normalizes host/port, HTTPS and IPv6 without dropping non-root paths', () => {
    expect(normalizeServerUrl(' NAS.local:8081/ ')).toBe('http://nas.local:8081')
    expect(normalizeServerUrl('https://NAS.local:443/')).toBe('https://nas.local')
    expect(normalizeServerUrl('[::1]:8081')).toBe('http://[::1]:8081')
  })
  it.each(['', 'file:///tmp', 'ftp://nas.local', 'http://user:pass@host', 'http://host/library', 'http://host/?token=abc', 'http://host/#hash', 'http://host:99999'])('rejects unsafe or unsupported address %s', url => {
    expect(() => normalizeServerUrl(url)).toThrow()
  })
  it('persists deduplicated named entries and last connection across restarts', () => {
    const { file, store } = fixture()
    const first = store.save({ name: 'Home', url: 'nas.local:8081' })
    const duplicate = store.save({ name: 'Renamed', url: 'http://nas.local:8081/' })
    expect(duplicate.id).toBe(first.id)
    const second = store.save({ name: 'Office', url: 'https://office.local' })
    store.remember(second.id)
    const reopened = new ServerConnectionStore(file)
    expect(reopened.snapshot()).toEqual({ schema: 1, servers: [duplicate, second], lastServerId: second.id })
    reopened.remove(second.id)
    expect(new ServerConnectionStore(file).snapshot()).toEqual({ schema: 1, servers: [duplicate] })
  })
  it('rejects duplicate edits and invalid input without overwriting disk', () => {
    const { store, file } = fixture()
    const first = store.save({ name: 'A', url: 'a.local' })
    store.save({ name: 'B', url: 'b.local' })
    const before = readFileSync(file, 'utf8')
    expect(() => store.save({ ...first, url: 'b.local' })).toThrow()
    expect(() => store.save({ name: ' ', url: 'c.local' })).toThrow()
    expect(() => store.remember('missing')).toThrow()
    expect(readFileSync(file, 'utf8')).toBe(before)
  })
  it('preserves corrupt files and does not silently reset connections', () => {
    const { file } = fixture()
    writeFileSync(file, 'broken data')
    expect(() => new ServerConnectionStore(file)).toThrow()
    expect(readFileSync(file, 'utf8')).toBe('broken data')
  })
  it('isolates cookies and storage even between ports on the same hostname', () => {
    expect(serverSessionPartition('http://nas.local:8081')).not.toBe(serverSessionPartition('http://nas.local:8082'))
    expect(serverSessionPartition('https://nas.local')).not.toBe(serverSessionPartition('http://nas.local'))
    expect(serverSessionPartition('HTTP://NAS.local:80/')).toBe(serverSessionPartition('http://nas.local'))
  })
  it('probes Curated identity without cookies and disallows redirects', async () => {
    const fetcher = vi.fn<typeof fetch>().mockResolvedValue(new Response(JSON.stringify({ name: 'curated', version: '1.5.7' })))
    await probeServer('nas.local:8081', fetcher)
    expect(fetcher).toHaveBeenCalledWith('http://nas.local:8081/api/health', expect.objectContaining({ redirect: 'error', credentials: 'omit', signal: expect.any(AbortSignal) }))
  })
  it('rejects other apps, oversized health payloads and failed requests', async () => {
    for (const response of [new Response('{"name":"other"}'), new Response('x'.repeat(65537)), new Response('', { status: 503 })]) {
      await expect(probeServer('nas.local', vi.fn<typeof fetch>().mockResolvedValue(response))).rejects.toThrow()
    }
    await expect(probeServer('nas.local', vi.fn<typeof fetch>().mockRejectedValue(new Error('offline')))).rejects.toThrow('无法连接服务器')
  })
})

it('upgrades a released 0.1.0 profile without changing IDs, names, last selection or its origin session', () => {
  const { file } = fixture()
  const prior = { schema: 1, servers: [{ id: 'original-id', name: 'My NAS', url: 'http://nas.local:8081' }], lastServerId: 'original-id' }
  writeFileSync(file, JSON.stringify(prior))
  const store = new ServerConnectionStore(file)
  expect(store.snapshot()).toEqual(prior)
  const partition = serverSessionPartition(prior.servers[0].url)
  store.bindIdentity('original-id', '18a31111-36e0-424b-9f9f-d1e9cfb74ed2', partition)
  store.save({ ...prior.servers[0], name: 'Renamed' })
  expect(new ServerConnectionStore(file).snapshot()).toMatchObject({ lastServerId: 'original-id', servers: [{ id: 'original-id', name: 'Renamed', partition }] })
})

it('imports development connections atomically and keeps the original file and identity session', async () => {
  const { connectionPartition } = await import('./connections')
  const { file } = fixture()
  const previousFile = path.join(path.dirname(file), 'connections.json')
  const connection = { url: 'http://nas.local:8081', name: 'NAS', serverId: '18a31111-36e0-424b-9f9f-d1e9cfb74ed2' }
  const previous = JSON.stringify({ version: 1, connections: [connection], lastUrl: connection.url })
  writeFileSync(previousFile, previous)
  const state = new ServerConnectionStore(file).snapshot()
  expect(state.servers[0]).toMatchObject({ ...connection, partition: connectionPartition(connection) })
  expect(state.lastServerId).toBe(state.servers[0].id)
  expect(readFileSync(previousFile, 'utf8')).toBe(previous)
  writeFileSync(previousFile, 'invalid')
  expect(new ServerConnectionStore(file).snapshot()).toEqual(state)
})

it('does not persist a partial development migration when a later record is corrupt', async () => {
  const { existsSync } = await import('node:fs')
  const { file } = fixture()
  const previousFile = path.join(path.dirname(file), 'connections.json')
  const prior = JSON.stringify({ version: 1, connections: [{ url: 'http://nas.local', name: 'NAS', serverId: 'old' }, { url: 'bad/path', name: 'bad', serverId: 'old2' }] })
  writeFileSync(previousFile, prior)
  expect(() => new ServerConnectionStore(file)).toThrow()
  expect(existsSync(file)).toBe(false)
  expect(readFileSync(previousFile, 'utf8')).toBe(prior)
})
