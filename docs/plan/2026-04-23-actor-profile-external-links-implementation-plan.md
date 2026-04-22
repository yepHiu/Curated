# Actor Profile External Links Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Web API backed, user-managed external links to the actor profile card while keeping the scraped `homepage` field unchanged.

**Architecture:** Store actor external links in a dedicated SQLite table keyed by `actor_id`, expose them through the existing actor profile read path, and update them via a dedicated PATCH endpoint that replaces the full list. Keep the frontend scope narrow: wire the API directly into `ActorProfileCard.vue`, reuse the existing actor profile state, and explicitly avoid expanding Mock-mode service contracts.

**Tech Stack:** Go, SQLite migrations, Vue 3, TypeScript, Vitest, vue-test-utils, vue-i18n

---

## File Map

- Create: `backend/internal/storage/migrations/0017_actor_external_links.sql`
- Create: `backend/internal/storage/actor_external_links_repository.go`
- Modify: `backend/internal/contracts/contracts.go`
- Modify: `backend/internal/storage/actor_repository.go`
- Modify: `backend/internal/storage/actor_repository_test.go`
- Modify: `backend/internal/server/server.go`
- Modify: `backend/internal/server/server_test.go`
- Modify: `src/api/types.ts`
- Modify: `src/api/endpoints.ts`
- Create: `src/lib/actor-external-links.ts`
- Create: `src/lib/actor-external-links.test.ts`
- Modify: `src/components/jav-library/ActorProfileCard.vue`
- Create: `src/components/jav-library/ActorProfileCard.test.ts`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/en.json`
- Modify: `src/locales/ja.json`
- Modify: `API.md`
- Modify: `.cursor/rules/project-facts.mdc`
- Modify: `CLAUDE.md`

Files intentionally left unchanged:

- `src/services/contracts/library-service.ts`
- `src/services/adapters/mock/mock-library-service.ts`
- `src/components/jav-library/ActorsPage.vue`

Reason: this feature is confirmed as Web API only and must be managed only inside the actor profile card.

### Task 1: Backend Schema And Storage

**Files:**
- Create: `backend/internal/storage/migrations/0017_actor_external_links.sql`
- Create: `backend/internal/storage/actor_external_links_repository.go`
- Modify: `backend/internal/contracts/contracts.go`
- Modify: `backend/internal/storage/actor_repository.go`
- Modify: `backend/internal/storage/actor_repository_test.go`

- [ ] **Step 1: Write the failing storage tests**

Append these tests to `backend/internal/storage/actor_repository_test.go`:

```go
func TestGetActorProfile_ExternalLinks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "actor-links.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-1",
		Path:     filepath.Join(root, "MOV-1.mp4"),
		FileName: "MOV-1.mp4",
		Number:   "MOV-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID:        outcome.MovieID,
		Number:         "MOV-1",
		Title:          "One",
		Summary:        "s",
		Studio:         "Studio",
		Actors:         []string{"Alpha Star"},
		RuntimeMinutes: 1,
	}); err != nil {
		t.Fatal(err)
	}

	if err := store.ReplaceActorExternalLinksByName(ctx, "Alpha Star", []string{
		" https://example.com/a ",
		"https://example.com/b",
		"https://example.com/a",
	}); err != nil {
		t.Fatal(err)
	}

	profile, err := store.GetActorProfile(ctx, "Alpha Star")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"https://example.com/a", "https://example.com/b"}
	if !reflect.DeepEqual(profile.ExternalLinks, want) {
		t.Fatalf("external links = %#v, want %#v", profile.ExternalLinks, want)
	}
}

func TestReplaceActorExternalLinksByName_RejectsInvalidURL(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(root, "actor-links-invalid.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-2",
		Path:     filepath.Join(root, "MOV-2.mp4"),
		FileName: "MOV-2.mp4",
		Number:   "MOV-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID:        outcome.MovieID,
		Number:         "MOV-2",
		Title:          "Two",
		Summary:        "s",
		Studio:         "Studio",
		Actors:         []string{"Beta Star"},
		RuntimeMinutes: 1,
	}); err != nil {
		t.Fatal(err)
	}

	err = store.ReplaceActorExternalLinksByName(ctx, "Beta Star", []string{"javascript:alert(1)"})
	if !errors.Is(err, ErrInvalidActorExternalLinks) {
		t.Fatalf("err = %v, want ErrInvalidActorExternalLinks", err)
	}
}
```

- [ ] **Step 2: Run the storage tests to verify they fail**

Run:

```bash
cd backend
go test ./internal/storage -run "Test(GetActorProfile_ExternalLinks|ReplaceActorExternalLinksByName_RejectsInvalidURL)" -count=1
```

Expected:

- FAIL because `ReplaceActorExternalLinksByName`
- FAIL because `profile.ExternalLinks`
- FAIL because `ErrInvalidActorExternalLinks`

- [ ] **Step 3: Write the migration and minimal storage implementation**

Create `backend/internal/storage/migrations/0017_actor_external_links.sql`:

```sql
CREATE TABLE IF NOT EXISTS actor_external_links (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor_id INTEGER NOT NULL,
  url TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (actor_id) REFERENCES actors(id) ON DELETE CASCADE,
  UNIQUE(actor_id, url)
);

CREATE INDEX IF NOT EXISTS idx_actor_external_links_actor_sort
  ON actor_external_links(actor_id, sort_order, id);
```

Add to `backend/internal/contracts/contracts.go`:

```go
type ActorProfileDTO struct {
	Name             string   `json:"name"`
	AvatarURL        string   `json:"avatarUrl,omitempty"`
	AvatarRemoteURL  string   `json:"avatarRemoteUrl,omitempty"`
	AvatarLocalURL   string   `json:"avatarLocalUrl,omitempty"`
	HasLocalAvatar   bool     `json:"hasLocalAvatar,omitempty"`
	Summary          string   `json:"summary,omitempty"`
	Homepage         string   `json:"homepage,omitempty"`
	Provider         string   `json:"provider,omitempty"`
	ProviderActorID  string   `json:"providerActorId,omitempty"`
	Height           int      `json:"height,omitempty"`
	Birthday         string   `json:"birthday,omitempty"`
	ProfileUpdatedAt string   `json:"profileUpdatedAt,omitempty"`
	UserTags         []string `json:"userTags,omitempty"`
	ExternalLinks    []string `json:"externalLinks,omitempty"`
}
```

Create `backend/internal/storage/actor_external_links_repository.go`:

```go
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"curated-backend/internal/contracts"
)

var ErrInvalidActorExternalLinks = errors.New("invalid actor external links")

const maxActorExternalLinks = 16
const maxActorExternalLinkLength = 2048

func NormalizeActorExternalLinksForPatch(raw []string) ([]string, error) {
	if raw == nil {
		raw = []string{}
	}
	if len(raw) > maxActorExternalLinks {
		return nil, fmt.Errorf("%w: at most %d links per actor", ErrInvalidActorExternalLinks, maxActorExternalLinks)
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if len(item) > maxActorExternalLinkLength {
			return nil, fmt.Errorf("%w: url longer than %d characters", ErrInvalidActorExternalLinks, maxActorExternalLinkLength)
		}
		parsed, err := url.ParseRequestURI(item)
		if err != nil || parsed == nil || parsed.Host == "" {
			return nil, fmt.Errorf("%w: invalid url %q", ErrInvalidActorExternalLinks, item)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return nil, fmt.Errorf("%w: unsupported scheme %q", ErrInvalidActorExternalLinks, parsed.Scheme)
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out, nil
}

func (s *SQLiteStore) loadActorExternalLinksForIDs(ctx context.Context, ids []int64) (map[int64][]string, error) {
	out := make(map[int64][]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT actor_id, url FROM actor_external_links WHERE actor_id IN (%s) ORDER BY actor_id, sort_order, id`,
		placeholders,
	), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var actorID int64
		var item string
		if err := rows.Scan(&actorID, &item); err != nil {
			return nil, err
		}
		out[actorID] = append(out[actorID], item)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ReplaceActorExternalLinksByName(ctx context.Context, name string, rawLinks []string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return contracts.ErrActorNotFound
	}
	links, err := NormalizeActorExternalLinksForPatch(rawLinks)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var actorID int64
	switch err := tx.QueryRowContext(ctx, `SELECT id FROM actors WHERE name = ?`, name).Scan(&actorID); {
	case errors.Is(err, sql.ErrNoRows):
		return contracts.ErrActorNotFound
	case err != nil:
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM actor_external_links WHERE actor_id = ?`, actorID); err != nil {
		return err
	}
	for i, item := range links {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO actor_external_links (actor_id, url, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			actorID, item, i, nowUTC(), nowUTC(),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
```

Update `backend/internal/storage/actor_repository.go` inside `GetActorProfile`:

```go
	linksByID, err := s.loadActorExternalLinksForIDs(ctx, []int64{actorID})
	if err != nil {
		return contracts.ActorProfileDTO{}, err
	}
	dto.ExternalLinks = linksByID[actorID]
```

- [ ] **Step 4: Run the storage tests to verify they pass**

Run:

```bash
cd backend
go test ./internal/storage -run "Test(GetActorProfile_ExternalLinks|ReplaceActorExternalLinksByName_RejectsInvalidURL)" -count=1
```

Expected:

- PASS

- [ ] **Step 5: Commit the schema and storage slice**

Run:

```bash
git add backend/internal/storage/migrations/0017_actor_external_links.sql backend/internal/storage/actor_external_links_repository.go backend/internal/contracts/contracts.go backend/internal/storage/actor_repository.go backend/internal/storage/actor_repository_test.go
git commit -m "feat: persist actor external links"
```

### Task 2: Backend API Contract And Handler

**Files:**
- Modify: `backend/internal/contracts/contracts.go`
- Modify: `backend/internal/server/server.go`
- Modify: `backend/internal/server/server_test.go`

- [ ] **Step 1: Write the failing handler tests**

Append these tests to `backend/internal/server/server_test.go`:

```go
func TestHandlePatchActorExternalLinks_Success(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "handler-actor-links.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "task-actor-links",
		Path:     filepath.Join(root, "MOV-3.mp4"),
		FileName: "MOV-3.mp4",
		Number:   "MOV-3",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
		MovieID:        outcome.MovieID,
		Number:         "MOV-3",
		Title:          "Three",
		Summary:        "s",
		Studio:         "Studio",
		Actors:         []string{"Gamma Star"},
		RuntimeMinutes: 1,
	}); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/library/actors/external-links?name=Gamma%20Star", strings.NewReader(`{"externalLinks":["https://example.com/a","https://example.com/b"]}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body contracts.ActorProfileDTO
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	want := []string{"https://example.com/a", "https://example.com/b"}
	if !reflect.DeepEqual(body.ExternalLinks, want) {
		t.Fatalf("external links = %#v, want %#v", body.ExternalLinks, want)
	}
}

func TestHandlePatchActorExternalLinks_InvalidURL(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := storage.NewSQLiteStore(filepath.Join(root, "handler-actor-links-invalid.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{Cfg: config.Config{}, Logger: zap.NewNop(), Store: store})
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/library/actors/external-links?name=Nobody", strings.NewReader(`{"externalLinks":["ftp://example.com"]}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 400 or 404", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Run the handler tests to verify they fail**

Run:

```bash
cd backend
go test ./internal/server -run "TestHandlePatchActorExternalLinks_(Success|InvalidURL)" -count=1
```

Expected:

- FAIL because the route does not exist
- or FAIL because `PatchActorExternalLinksBody` is undefined

- [ ] **Step 3: Add the DTO, route, and handler**

Add to `backend/internal/contracts/contracts.go`:

```go
type PatchActorExternalLinksBody struct {
	ExternalLinks []string `json:"externalLinks"`
}
```

Register the route in `backend/internal/server/server.go` near the existing actor routes:

```go
mux.HandleFunc("PATCH /api/library/actors/external-links", h.handlePatchActorExternalLinks)
```

Implement the handler in `backend/internal/server/server.go`:

```go
func (h *Handler) handlePatchActorExternalLinks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "name is required")
		return
	}

	body, err := io.ReadAll(r.Body)
	if r.Body != nil {
		_ = r.Body.Close()
	}
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "failed to read body")
		return
	}

	var in contracts.PatchActorExternalLinksBody
	if err := json.Unmarshal(body, &in); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
		return
	}

	if err := h.store.ReplaceActorExternalLinksByName(r.Context(), name, in.ExternalLinks); err != nil {
		if errors.Is(err, contracts.ErrActorNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "actor not found")
			return
		}
		if errors.Is(err, storage.ErrInvalidActorExternalLinks) {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
			return
		}
		if h.logger != nil {
			h.logger.Warn("patch actor external links failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to update actor external links")
		return
	}

	profile, err := h.store.GetActorProfile(r.Context(), name)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("load actor after external links patch failed", zap.Error(err))
		}
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load actor profile")
		return
	}
	h.enrichActorProfileLocalAvatar(r.Context(), &profile)
	writeJSON(w, http.StatusOK, profile)
}
```

- [ ] **Step 4: Run the handler tests to verify they pass**

Run:

```bash
cd backend
go test ./internal/server -run "TestHandlePatchActorExternalLinks_(Success|InvalidURL)" -count=1
```

Expected:

- PASS

- [ ] **Step 5: Commit the HTTP slice**

Run:

```bash
git add backend/internal/contracts/contracts.go backend/internal/server/server.go backend/internal/server/server_test.go
git commit -m "feat: add actor external links api"
```

### Task 3: Frontend API Types And URL Validation Helper

**Files:**
- Modify: `src/api/types.ts`
- Modify: `src/api/endpoints.ts`
- Create: `src/lib/actor-external-links.ts`
- Create: `src/lib/actor-external-links.test.ts`

- [ ] **Step 1: Write the failing helper tests**

Create `src/lib/actor-external-links.test.ts`:

```ts
import { describe, expect, it } from "vitest"

import {
  isValidActorExternalLink,
  normalizeActorExternalLinkDraft,
} from "./actor-external-links"

describe("actor external links", () => {
  it("trims surrounding whitespace before save", () => {
    expect(normalizeActorExternalLinkDraft("  https://example.com/a  ")).toBe("https://example.com/a")
  })

  it("accepts absolute http and https urls", () => {
    expect(isValidActorExternalLink("http://example.com")).toBe(true)
    expect(isValidActorExternalLink("https://example.com/path")).toBe(true)
  })

  it("rejects relative and non-http urls", () => {
    expect(isValidActorExternalLink("/actor/1")).toBe(false)
    expect(isValidActorExternalLink("ftp://example.com")).toBe(false)
    expect(isValidActorExternalLink("javascript:alert(1)")).toBe(false)
  })
})
```

- [ ] **Step 2: Run the helper tests to verify they fail**

Run:

```bash
pnpm test -- src/lib/actor-external-links.test.ts
```

Expected:

- FAIL because `src/lib/actor-external-links.ts` does not exist

- [ ] **Step 3: Implement the helper and API seam**

Create `src/lib/actor-external-links.ts`:

```ts
export const MAX_ACTOR_EXTERNAL_LINKS = 16

export function normalizeActorExternalLinkDraft(raw: string): string {
  return raw.trim()
}

export function isValidActorExternalLink(raw: string): boolean {
  try {
    const parsed = new URL(raw)
    return parsed.protocol === "http:" || parsed.protocol === "https:"
  } catch {
    return false
  }
}
```

Update `src/api/types.ts`:

```ts
export interface ActorProfileDTO {
  name: string
  avatarUrl?: string
  avatarRemoteUrl?: string
  avatarLocalUrl?: string
  hasLocalAvatar?: boolean
  summary?: string
  homepage?: string
  provider?: string
  providerActorId?: string
  height?: number
  birthday?: string
  profileUpdatedAt?: string
  userTags?: string[]
  externalLinks?: string[]
}

export interface PatchActorExternalLinksBody {
  externalLinks: string[]
}
```

Update `src/api/endpoints.ts`:

```ts
import type {
  ActorListItemDTO,
  ActorProfileDTO,
  ActorsListDTO,
  PatchActorExternalLinksBody,
  // ...
} from "./types"

patchActorExternalLinks(name: string, externalLinks: string[]): Promise<ActorProfileDTO> {
  const q = new URLSearchParams({ name })
  const body: PatchActorExternalLinksBody = { externalLinks }
  return httpClient.patch<ActorProfileDTO>(`/library/actors/external-links?${q.toString()}`, body)
},
```

- [ ] **Step 4: Run the helper tests to verify they pass**

Run:

```bash
pnpm test -- src/lib/actor-external-links.test.ts
```

Expected:

- PASS

- [ ] **Step 5: Commit the frontend API seam**

Run:

```bash
git add src/api/types.ts src/api/endpoints.ts src/lib/actor-external-links.ts src/lib/actor-external-links.test.ts
git commit -m "feat: add frontend actor external link api"
```

### Task 4: Actor Profile Card UI, Locales, And Component Tests

**Files:**
- Modify: `src/components/jav-library/ActorProfileCard.vue`
- Create: `src/components/jav-library/ActorProfileCard.test.ts`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/en.json`
- Modify: `src/locales/ja.json`

- [ ] **Step 1: Write the failing component tests**

Create `src/components/jav-library/ActorProfileCard.test.ts`:

```ts
import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"

import ActorProfileCard from "./ActorProfileCard.vue"

const getActorProfile = vi.fn()
const patchActorExternalLinks = vi.fn()

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("vue-router", () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
}))

vi.mock("@vueuse/core", () => ({
  useFocusWithin: () => ({ focused: ref(true) }),
  useResizeObserver: vi.fn(),
  onClickOutside: vi.fn(),
}))

vi.mock("@/api/endpoints", () => ({
  api: {
    getActorProfile,
    patchActorExternalLinks,
    scrapeActorProfile: vi.fn(),
    getTaskStatus: vi.fn(),
  },
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    patchActorUserTags: vi.fn(),
  }),
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: vi.fn(),
}))

vi.mock("@/components/ui/avatar", () => ({
  Avatar: { template: "<div><slot /></div>" },
  AvatarFallback: { template: "<div><slot /></div>" },
  AvatarImage: { template: "<img />" },
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: { template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    emits: ["click"],
    template: "<button @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/card", () => ({
  Card: { template: "<div><slot /></div>" },
  CardContent: { template: "<div><slot /></div>" },
  CardHeader: { template: "<div><slot /></div>" },
  CardTitle: { template: "<div><slot /></div>" },
}))

describe("ActorProfileCard", () => {
  beforeEach(() => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    getActorProfile.mockReset()
    patchActorExternalLinks.mockReset()
  })

  it("renders saved links and adds a valid external link", async () => {
    getActorProfile.mockResolvedValue({
      name: "Alpha Star",
      externalLinks: ["https://example.com/a"],
      userTags: [],
    })
    patchActorExternalLinks.mockResolvedValue({
      name: "Alpha Star",
      externalLinks: ["https://example.com/a", "https://example.com/b"],
      userTags: [],
    })

    const wrapper = mount(ActorProfileCard, {
      props: {
        actorName: "Alpha Star",
        userTagSuggestions: [],
      },
      global: {
        stubs: {
          Teleport: false,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain("https://example.com/a")

    await wrapper.get("[data-actor-external-link-add]").trigger("click")
    await wrapper.get("[data-actor-external-link-input]").setValue("https://example.com/b")
    await wrapper.get("[data-actor-external-link-add]").trigger("click")
    await flushPromises()

    expect(patchActorExternalLinks).toHaveBeenCalledWith("Alpha Star", [
      "https://example.com/a",
      "https://example.com/b",
    ])
    expect(wrapper.text()).toContain("https://example.com/b")
  })

  it("shows an inline error for invalid external links", async () => {
    getActorProfile.mockResolvedValue({
      name: "Alpha Star",
      externalLinks: [],
      userTags: [],
    })

    const wrapper = mount(ActorProfileCard, {
      props: {
        actorName: "Alpha Star",
        userTagSuggestions: [],
      },
    })

    await flushPromises()

    await wrapper.get("[data-actor-external-link-add]").trigger("click")
    await wrapper.get("[data-actor-external-link-input]").setValue("ftp://example.com")
    await wrapper.get("[data-actor-external-link-add]").trigger("click")

    expect(wrapper.text()).toContain("library.actorExternalLinksInvalid")
    expect(patchActorExternalLinks).not.toHaveBeenCalled()
  })
})
```

- [ ] **Step 2: Run the component tests to verify they fail**

Run:

```bash
pnpm test -- src/components/jav-library/ActorProfileCard.test.ts
```

Expected:

- FAIL because `patchActorExternalLinks` is not implemented in `api`
- FAIL because the component has no external link UI or test selectors

- [ ] **Step 3: Implement the profile card UI and translations**

Add to `src/components/jav-library/ActorProfileCard.vue`:

```ts
import {
  MAX_ACTOR_EXTERNAL_LINKS,
  isValidActorExternalLink,
  normalizeActorExternalLinkDraft,
} from "@/lib/actor-external-links"

const externalLinksSaving = ref(false)
const externalLinkInputOpen = ref(false)
const newExternalLinkDraft = ref("")
const externalLinkFormError = ref("")

const externalLinks = computed(() => profile.value?.externalLinks ?? [])

async function patchActorExternalLinks(next: string[]) {
  const name = props.actorName.trim()
  if (!name) {
    return
  }
  externalLinksSaving.value = true
  externalLinkFormError.value = ""
  try {
    profile.value = await api.patchActorExternalLinks(name, next)
  } catch (err) {
    externalLinkFormError.value = err instanceof Error ? err.message : t("library.actorExternalLinksSaveError")
  } finally {
    externalLinksSaving.value = false
  }
}

async function addExternalLink() {
  const next = normalizeActorExternalLinkDraft(newExternalLinkDraft.value)
  if (!next) {
    return
  }
  if (!isValidActorExternalLink(next)) {
    externalLinkFormError.value = t("library.actorExternalLinksInvalid")
    return
  }
  if (externalLinks.value.includes(next)) {
    newExternalLinkDraft.value = ""
    return
  }
  if (externalLinks.value.length >= MAX_ACTOR_EXTERNAL_LINKS) {
    externalLinkFormError.value = t("library.actorExternalLinksMaxCount", { n: MAX_ACTOR_EXTERNAL_LINKS })
    return
  }
  await patchActorExternalLinks([...externalLinks.value, next])
  newExternalLinkDraft.value = ""
}

async function removeExternalLink(link: string) {
  await patchActorExternalLinks(externalLinks.value.filter((item) => item !== link))
}
```

Render the block in `src/components/jav-library/ActorProfileCard.vue` below the homepage row:

```vue
<section class="space-y-2">
  <div class="flex items-center justify-between gap-2">
    <p class="text-sm text-muted-foreground">{{ t("library.actorExternalLinks") }}</p>
    <Button
      data-actor-external-link-add
      type="button"
      variant="secondary"
      class="rounded-2xl"
      :disabled="externalLinksSaving"
      @click="externalLinkInputOpen ? addExternalLink() : (externalLinkInputOpen = true)"
    >
      <Plus data-icon="inline-start" />
      {{ t("common.add") }}
    </Button>
  </div>

  <div v-if="externalLinkInputOpen" class="flex items-center gap-2 rounded-2xl border border-border/80 bg-background/80 px-3 py-1.5 shadow-sm">
    <input
      data-actor-external-link-input
      v-model="newExternalLinkDraft"
      type="url"
      inputmode="url"
      autocomplete="off"
      :disabled="externalLinksSaving"
      :placeholder="t('library.actorExternalLinksPlaceholder')"
      class="h-8 min-w-0 flex-1 border-0 bg-transparent px-0 text-sm shadow-none outline-none focus-visible:ring-0"
      @keydown.enter.prevent="addExternalLink"
    />
    <Button type="button" variant="ghost" size="icon" @click="externalLinkInputOpen = false">
      <X class="size-4" />
    </Button>
  </div>

  <ul v-if="externalLinks.length > 0" class="space-y-2">
    <li
      v-for="link in externalLinks"
      :key="link"
      data-actor-external-link-row
      class="flex items-center gap-2 rounded-2xl border border-border/60 bg-background/60 px-3 py-2"
    >
      <a
        class="min-w-0 flex-1 truncate text-primary underline-offset-4 hover:underline"
        :href="link"
        target="_blank"
        rel="noopener noreferrer"
      >
        {{ link }}
      </a>
      <button
        :data-actor-external-link-remove="link"
        type="button"
        class="inline-flex size-7 shrink-0 items-center justify-center rounded-full text-muted-foreground transition hover:bg-destructive/15 hover:text-destructive"
        :disabled="externalLinksSaving"
        @click="removeExternalLink(link)"
      >
        <X class="size-4" />
      </button>
    </li>
  </ul>

  <p v-if="externalLinkFormError" class="text-sm text-destructive">
    {{ externalLinkFormError }}
  </p>
</section>
```

Add locale strings to `src/locales/zh-CN.json`, `src/locales/en.json`, and `src/locales/ja.json` under `library`:

```json
{
  "actorExternalLinks": "外链",
  "actorExternalLinksPlaceholder": "https://example.com/profile",
  "actorExternalLinksInvalid": "请输入有效的 http:// 或 https:// 链接",
  "actorExternalLinksSaveError": "保存演员外链失败",
  "actorExternalLinksMaxCount": "最多可添加 {n} 个外链"
}
```

Use these exact per-locale values when editing the real locale files:

`src/locales/zh-CN.json`

```json
{
  "actorExternalLinks": "外链",
  "actorExternalLinksPlaceholder": "https://example.com/profile",
  "actorExternalLinksInvalid": "请输入有效的 http:// 或 https:// 链接",
  "actorExternalLinksSaveError": "保存演员外链失败",
  "actorExternalLinksMaxCount": "最多可添加 {n} 个外链"
}
```

`src/locales/en.json`

```json
{
  "actorExternalLinks": "External Links",
  "actorExternalLinksPlaceholder": "https://example.com/profile",
  "actorExternalLinksInvalid": "Enter a valid http:// or https:// link",
  "actorExternalLinksSaveError": "Failed to save actor external links",
  "actorExternalLinksMaxCount": "You can add up to {n} external links"
}
```

`src/locales/ja.json`

```json
{
  "actorExternalLinks": "外部リンク",
  "actorExternalLinksPlaceholder": "https://example.com/profile",
  "actorExternalLinksInvalid": "有効な http:// または https:// リンクを入力してください",
  "actorExternalLinksSaveError": "出演者の外部リンクを保存できませんでした",
  "actorExternalLinksMaxCount": "外部リンクは最大 {n} 件まで追加できます"
}
```

- [ ] **Step 4: Run the component tests to verify they pass**

Run:

```bash
pnpm test -- src/components/jav-library/ActorProfileCard.test.ts
```

Expected:

- PASS

- [ ] **Step 5: Commit the UI slice**

Run:

```bash
git add src/components/jav-library/ActorProfileCard.vue src/components/jav-library/ActorProfileCard.test.ts src/locales/zh-CN.json src/locales/en.json src/locales/ja.json
git commit -m "feat: manage actor external links in profile card"
```

### Task 5: Documentation Sync And Final Verification

**Files:**
- Modify: `API.md`
- Modify: `.cursor/rules/project-facts.mdc`
- Modify: `CLAUDE.md`

- [ ] **Step 1: Update API and project memory docs**

Update `API.md` with the new endpoint:

````md
### PATCH `/api/library/actors/external-links?name=`

Replace the full user-managed external link list for the actor identified by exact display name.

Request body:

```json
{
  "externalLinks": [
    "https://example.com/a",
    "https://example.com/b"
  ]
}
````

Response: `200 OK` with the refreshed `ActorProfileDTO`.
```

Update `.cursor/rules/project-facts.mdc` in the actor API section:

```md
- `PATCH /api/library/actors/external-links?name=`: replace actor-scoped user-managed external links; distinct from scraped `homepage`; returned by `GET /api/library/actors/profile`.
```

Update `CLAUDE.md` in the API list with the same endpoint summary.

- [ ] **Step 2: Run the focused verification commands**

Run:

```bash
pnpm test -- src/lib/actor-external-links.test.ts src/components/jav-library/ActorProfileCard.test.ts
```

Run:

```bash
pnpm typecheck
```

Run:

```bash
cd backend
go test ./internal/storage ./internal/server
```

Expected:

- all Vitest cases PASS
- `pnpm typecheck` exits 0
- backend storage and server packages PASS

- [ ] **Step 3: Run the broader safety checks**

Run:

```bash
pnpm lint
```

Run:

```bash
cd backend
go test ./...
```

Expected:

- lint exits 0
- full Go test suite exits 0

- [ ] **Step 4: Commit docs and verification-backed finish**

Run:

```bash
git add API.md .cursor/rules/project-facts.mdc CLAUDE.md
git commit -m "docs: document actor external links api"
```

- [ ] **Step 5: Prepare the execution handoff**

Use one of these execution modes after this plan is approved:

```text
1. Subagent-Driven (recommended): use superpowers:subagent-driven-development
2. Inline Execution: use superpowers:executing-plans
```

## Self-Review

- Spec coverage: data model, PATCH endpoint, actor profile DTO extension, ActorProfileCard-only UI, URL validation, and Web API only scope are all covered by Tasks 1-4. Endpoint and memory docs are covered by Task 5.
- Placeholder scan: no `TODO`, `TBD`, or vague “handle appropriately” language remains in the task steps.
- Type consistency: the plan uses `externalLinks` consistently across SQLite, Go DTOs, JSON payloads, TypeScript DTOs, the frontend helper, and the component state.
