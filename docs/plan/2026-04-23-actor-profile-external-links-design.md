# Actor Profile External Links Design

Date: 2026-04-23
Status: Approved

## Goal

Add a user-managed external links feature in the actor profile card.

Scope confirmed:

- Keep the existing scraped `homepage` field unchanged.
- Add a separate user-managed external links list for each actor.
- Each item is URL-only. No title scraping, no custom label.
- Management UI lives only in `ActorProfileCard.vue`.
- Support only `VITE_USE_WEB_API=true` for now. Mock mode is out of scope.

## Existing Context

- Actor profile read path already exists via `GET /api/library/actors/profile?name=`.
- Actor profile card already mixes direct profile API reads with user-managed actor tags.
- Scraped actor data is stored in `actors`.
- User-managed actor tags are stored separately in `actor_user_tags`.

This feature should follow the same separation principle: scraped fields remain scraped, user-managed links remain independent.

## Approaches

### Approach A: Dedicated `actor_external_links` table with full-list replacement PATCH

Add a new normalized table keyed by `actor_id`, return the links on profile reads, and update them through a dedicated PATCH endpoint that replaces the whole list.

Pros:

- Clean separation from scraped actor fields.
- Easy to preserve manual ordering.
- Extensible later if we add link type, note, or pinning.
- Low risk of scraper overwriting user data.

Cons:

- Slightly more backend work than a single-column shortcut.

Recommendation: use this approach.

### Approach B: Add `user_external_links_json` to `actors`

Store the whole user-managed URL list as JSON in the `actors` row.

Pros:

- Smallest schema and query surface.
- Faster to wire initially.

Cons:

- Mixes scraped and user-managed state in the same table.
- Harder to evolve for ordering metadata or future per-link fields.
- Makes the actor row do two unrelated jobs.

### Approach C: Reuse an existing generic user-tag style table

Encode links into an existing user metadata or tag mechanism.

Pros:

- Minimal schema change if we force-fit it.

Cons:

- Semantically wrong.
- Hard to validate as URLs.
- Hard to preserve clean UI/API contracts.

Not recommended.

## Recommended Design

### 1. Data model

Add a new SQLite table:

- `actor_external_links`
- Columns:
  - `id INTEGER PRIMARY KEY`
  - `actor_id INTEGER NOT NULL`
  - `url TEXT NOT NULL`
  - `sort_order INTEGER NOT NULL`
  - `created_at TEXT NOT NULL`
  - `updated_at TEXT NOT NULL`
- Constraints:
  - `FOREIGN KEY (actor_id) REFERENCES actors(id) ON DELETE CASCADE`
  - `UNIQUE(actor_id, url)`

Behavior:

- Links are user-managed only.
- Ordering is preserved using `sort_order`.
- `actors.homepage` remains the scraped homepage and is displayed separately.

### 2. API and contract

Extend actor profile DTOs with:

- `externalLinks?: string[]`

Add a dedicated endpoint:

- `PATCH /api/library/actors/external-links?name=`

Request body:

```json
{
  "externalLinks": [
    "https://example.com/profile",
    "https://social.example/actor/123"
  ]
}
```

Response:

- Return the refreshed `ActorProfileDTO`.

Why return full profile instead of only the list:

- The feature lives only in actor profile card.
- The component already owns profile state.
- Returning the full profile avoids an extra refresh request after save.

### 3. Backend behavior

Storage layer:

- `GetActorProfile` loads `externalLinks` ordered by `sort_order`.
- Add a new replace method similar to actor tags, for example:
  - `ReplaceActorExternalLinksByName(ctx, name, rawLinks []string) error`

Validation rules:

- Trim whitespace.
- Drop empty items.
- Require absolute `http://` or `https://` URLs.
- Deduplicate after normalization.
- Reject oversized lists.

Suggested limits:

- Max links per actor: `16`
- Max URL length: `2048`

PATCH semantics:

- Full-list replacement, same style as actor user tags.
- Empty array means clear all manual links.

### 4. Frontend behavior

UI location:

- Add a new "External Links" block in `ActorProfileCard.vue`.
- Keep scraped `homepage` as its own row above or beside the new user-managed block.

Interaction model:

- Existing saved links render as clickable rows or chips with:
  - URL text
  - open-in-new-tab behavior
  - remove action
- Add an inline input plus "Add" action for one URL at a time.
- Save immediately when adding or removing, same as actor tags.

Important scope decisions:

- No management entry in `ActorsPage.vue` or actor list cards.
- No title fetching.
- No link preview.
- No drag-sort in this phase.

Implementation note:

- Because this feature is explicitly Web API only and `ActorProfileCard.vue` already reads profile data directly via `api`, the write path can also call `api.patchActorExternalLinks(...)` directly instead of expanding the mock service contract.

### 5. Error handling and UX

Frontend:

- Show inline validation error when URL is invalid.
- Disable add/remove actions while request is in flight.
- On save success, update local `profile` state from the returned DTO.

Backend:

- `400` for invalid URL payload.
- `404` when actor name does not exist.
- `500` for unexpected storage errors.

### 6. Testing

Backend tests:

- migration creates `actor_external_links`
- `GetActorProfile` returns links in `sort_order`
- replace endpoint validates and persists correctly
- invalid scheme / invalid URL returns `400`
- empty array clears links

Frontend tests:

- actor profile card renders external links when present
- add valid URL triggers PATCH and updates UI
- invalid URL shows inline error
- remove link triggers PATCH and removes UI item

## Out of Scope

- Mock-mode persistence
- Automatic title scraping
- Custom labels or notes
- Bulk import
- Editing links from actor list cards
- Merging user links into scraped `homepage`

## Confirmed UI Choice

- The actor profile card external links block uses a row list, not compact chips.
- Reason: URLs are long and readability is more important than density in this area.
