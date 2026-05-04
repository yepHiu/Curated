# Connected Clients Display — Feature Design

2026-05-03 · Status: Draft

## Motivation

Curated runs as a local HTTP server. Users may access it from:

- **The local machine** (localhost) — the host running the backend
- **Remote devices on the same LAN** — laptop, tablet, phone, another desktop
- **Different browsers on the same or different machines** — Chrome, Firefox, Safari, Edge

There is currently no visibility into *who is connected*. The Settings → Overview page is the natural place to show a live list of active/connected clients so the user can audit access at a glance.

## Scope

A new **Connected Clients** card in **Settings → Overview** that lists every distinct client that has made HTTP requests to the backend in the current process lifetime, with identifying metadata.

## What each client entry shows

| Field | Source | Example |
|---|---|---|
| **IP address** | `http.Request.RemoteAddr` | `192.168.1.5:52341` |
| **Hostname** (LAN only) | Reverse-DNS lookup of the IP, best-effort | `macbook-pro.lan` |
| **Device type** | Inferred from User-Agent OS + form-factor heuristics | `Desktop` / `Laptop` / `Mobile` / `Tablet` |
| **OS** | Parsed from User-Agent | `Windows 11`, `macOS 15`, `Android 14` |
| **Browser / Client** | Parsed from User-Agent | `Chrome 132`, `Safari 18`, `Firefox 135` |
| **Access kind** | `127.0.0.1` / `::1` → "Local", else "Remote" | `Local` / `Remote` |
| **First seen** | Server timestamp of first request from this client | `2026-05-03 14:22:08` |
| **Last seen** | Server timestamp of most recent request | `2026-05-03 15:01:33` |
| **Local machine flag** | IP matches a known local interface of the host OS | `This device` badge |

### About MAC address

MAC addresses are **not available over HTTP**. A remote client's MAC is not present in any HTTP header and is not visible beyond the local network segment. We intentionally omit MAC address entirely — even the server's own MAC adds little value and could be misleading.

## Architecture

### Backend

#### 1. Client tracker (in-memory)

A new package `backend/internal/clienttracker` maintains an in-memory registry of distinct clients. It is **not persisted** — the list resets on backend restart.

```
ClientKey = hash(remoteIP + userAgent)
```

Each unique `ClientKey` gets one entry, updated on every request:

```go
type ClientSnapshot struct {
    RemoteAddr  string    // original r.RemoteAddr (ip:port)
    IP          string    // parsed IP portion
    UserAgent   string    // raw r.User-Agent header
    FirstSeen   time.Time
    LastSeen    time.Time
    RequestCount int64
}
```

The tracker exposes:
- `Record(r *http.Request)` — called by middleware on every request
- `Snapshot() []ClientSnapshot` — returns a sorted (LastSeen desc) copy of current clients

#### 2. Middleware

A new middleware `WithClientTracking` wraps the router. It calls `clienttracker.Record(r)` for every HTTP request (including `/api/health`). It is registered early in the middleware chain, after access logging.

#### 3. User-Agent parser

Use a lightweight Go UA parser library (e.g. `ua-parser-go` or `user_agent`) to extract:
- Browser name + major version
- OS name + version
- Device type (desktop / mobile / tablet / bot)

Fallback heuristics in-process when the parser can't identify the UA.

#### 4. Hostname resolution (best-effort)

For LAN IPs (private ranges: `192.168.x.x`, `10.x.x.x`, `172.16-31.x.x`), attempt a reverse-DNS lookup with a short timeout (500ms). If no result or timeout, the hostname field is empty. Never attempt reverse-DNS for public IPs.

#### 5. Local-interface detection

On backend startup, enumerate the host machine's network interfaces (`net.Interfaces()`) and cache the set of local IPs. A client whose IP matches any local interface IP gets the "This device" badge.

#### 6. New endpoint

```
GET /api/connected-clients
```

Response:

```json
{
  "clients": [
    {
      "ip": "192.168.1.5",
      "port": 52341,
      "hostname": "macbook-pro.lan",
      "userAgent": "Mozilla/5.0 (Macintosh; ...) ... Chrome/132.0.0.0 ...",
      "browser": "Chrome",
      "browserVersion": "132",
      "os": "macOS",
      "osVersion": "15.2",
      "deviceType": "laptop",
      "accessKind": "remote",
      "isLocalMachine": false,
      "firstSeen": "2026-05-03T14:22:08Z",
      "lastSeen": "2026-05-03T15:01:33Z",
      "requestCount": 284
    }
  ]
}
```

`serverMac` field removed — MAC addresses are not collected or exposed.

#### 7. Health endpoint enhancement

Optionally add `clientCount` to `GET /api/health` so the frontend can show a badge without an extra request:

```json
{
  "name": "curated-dev",
  ...
  "connectedClients": 3
}
```

### Frontend

#### 1. New DTOs in `src/api/types.ts`

```typescript
export interface ConnectedClientDTO {
  ip: string
  port: number
  hostname: string
  userAgent: string
  browser: string
  browserVersion: string
  os: string
  osVersion: string
  deviceType: 'desktop' | 'laptop' | 'mobile' | 'tablet' | 'unknown'
  accessKind: 'local' | 'remote'
  isLocalMachine: boolean
  firstSeen: string  // ISO 8601
  lastSeen: string   // ISO 8601
  requestCount: number
}

export interface ConnectedClientsDTO {
  clients: ConnectedClientDTO[]
}
```

#### 2. New API method in `src/api/endpoints.ts`

```typescript
getConnectedClients(): Promise<ConnectedClientsDTO> {
  return httpClient.get<ConnectedClientsDTO>("/connected-clients")
}
```

#### 3. New section component

`src/components/jav-library/settings/SettingsConnectedClientsSection.vue`

- Renders as a card in the Overview tab, below the three stat cards and above the watch-time heatmap
- Shows a table or card list of connected clients
- Each row shows: device icon (based on deviceType), OS + browser, IP + hostname, access kind badge, "This device" badge, last seen relative time
- Auto-refreshes: polls `GET /api/connected-clients` every 60 seconds while the Overview tab is visible
- Empty state: "No remote clients connected" when it's only the local machine
- No MAC address is collected or displayed (unavailable over HTTP)

#### 4. LibraryService updates

- Add `connectedClients: ComputedRef<ConnectedClientsDTO | null>`
- Add `refreshConnectedClients(): Promise<void>`
- Wire into `refreshSettings()` or provide a standalone refresh

#### 5. i18n keys

Add locale strings under `settings.overview.connectedClients.*` in `src/locales/en.json`, `ja.json`, `zh-CN.json`.

## UI Placement

**Settings → Overview tab**, as a new card between the dashboard stat cards and the watch-time heatmap.

```
┌──────────────────────────────────────────┐
│  Dashboard Stats (3 cards)               │
│  ┌────────┐ ┌────────┐ ┌────────┐       │
│  │ Movies │ │  Tags  │ │ Frames │        │
│  └────────┘ └────────┘ └────────┘       │
├──────────────────────────────────────────┤
│  Connected Clients            (new card) │
│  ┌──────────────────────────────────────┐│
│  │ 💻 This device — Windows 11 · Chrome ││
│  │    127.0.0.1              Local      ││
│  │                                      ││
│  │ 💻 MacBook Pro — macOS 15 · Safari   ││
│  │    192.168.1.5 · mbp.lan   Remote    ││
│  │    Last seen 2 min ago               ││
│  │                                      ││
│  │ 📱 iPhone 15 — iOS 19 · Safari       ││
│  │    192.168.1.8             Remote    ││
│  │    Last seen 12 min ago              ││
│  └──────────────────────────────────────┘│
├──────────────────────────────────────────┤
│  Watch Time Heatmap                      │
│  ...                                     │
└──────────────────────────────────────────┘
```

## Behaviors & edge cases

1. **Process-lifetime only**: The client list is in-memory. A backend restart clears it. No persistence needed.
2. **Deduplication**: Same IP + same User-Agent = same client. Different browser on same machine = separate client entry. Same browser private/incognito window = same entry (UA does not differ).
3. **Localhost variants**: Both `127.0.0.1` and `[::1]` are treated as "Local" access. The local-machine detection also covers the host's LAN IP when the request comes from the same machine.
4. **Stale clients**: No explicit eviction. The list naturally resets on restart. If the list grows very long (e.g. 50+ unique UAs from a single IP), cap at 50 most-recently-seen entries.
5. **Privacy**: No data leaves the machine. The endpoint is on the local HTTP API (no external exposure). The server MAC is already visible to anyone on the same LAN segment via ARP.
6. **Polling**: Frontend polls every 60s when the Overview tab is active. Stop polling when switching to another tab. This avoids unnecessary requests.
7. **Non-browser clients**: If the User-Agent is not a known browser (e.g. `curl/8.x`, `python-requests`), show the raw UA string under a "Tool / Script" category with a terminal icon.

## Implementation order

| Step | Layer | Effort |
|---|---|---|
| 1. UA parser dependency + `clienttracker` package | Backend | Medium |
| 2. Middleware registration in server.go | Backend | Small |
| 3. `GET /api/connected-clients` endpoint | Backend | Small |
| 4. Health endpoint enhancement (clientCount) | Backend | Small |
| 5. DTOs + API method | Frontend | Small |
| 6. LibraryService wiring | Frontend | Small |
| 7. `SettingsConnectedClientsSection.vue` | Frontend | Medium |
| 8. I18n strings (en, ja, zh-CN) | Frontend | Small |
| 9. Integration testing | Both | Small |

## Open questions

1. **Reverse-DNS**: Worth the complexity? It adds a best-effort hostname that is nice-to-have. Could start without it and add later.
2. **Polling interval**: 60 seconds, since the Overview tab isn't open all the time and client connections don't change rapidly.
3. **Persistence**: Currently design says in-memory only (reset on restart). Any value in persisting to SQLite for historical view?
