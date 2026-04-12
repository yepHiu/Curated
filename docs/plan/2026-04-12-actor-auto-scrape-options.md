# Actor Auto Scrape Options

Date: 2026-04-12

## Goal

Clarify how Curated should support automatic actor-profile scraping beyond the current on-demand flow.

## Current State

- The backend exposes `POST /api/library/actors/scrape?name=...` and runs an async `scrape.actor` task.
- The frontend `ActorProfileCard` loads `GET /api/library/actors/profile`.
- When the current actor profile has neither avatar nor summary, the frontend auto-triggers scrape for that single actor and refreshes after completion.
- There is no backend-wide automatic batch actor scraping during scan or as a scheduled background workflow.

## Candidate Approaches

### Option 1: Missing-profile auto scrape when actor is encountered during movie scan

- After a movie scan/import updates actor relations, enqueue actor scrape only for actors that exist in library rows but still lack profile essentials.
- Keep it backend-driven and asynchronous.

Pros:

- Most automatic for users
- Data improves in the background as the library grows

Risks:

- Can create many outbound requests during large scans
- Needs dedupe / throttling / cooldown, otherwise repeated scans may spam the same actor

### Option 2: Lazy auto scrape on actor profile read

- Keep current behavior, but move the trigger from frontend heuristics into backend profile-read flow or a more central frontend orchestration path.
- Only scrape actors that users actually browse to.

Pros:

- Lowest network cost
- Minimal behavior change

Risks:

- Not truly “full automatic”
- Actors never viewed remain unscripted

### Option 3: User-controlled batch actor auto scrape toggle

- Add a setting such as `autoActorProfileScrape`.
- When enabled, scan/import can enqueue missing actor profile tasks with rate limiting.
- When disabled, keep the current lazy behavior only.

Pros:

- Best balance between automation and control
- Fits current settings-driven architecture

Risks:

- More implementation surface: setting DTO, persistence, UI, scheduling rules

## Recommendation

Prefer Option 3.

Reason:

- The repository already separates behavior behind persisted settings like `autoLibraryWatch`.
- Actor scraping is network- and provider-dependent, so it should be opt-in rather than silently always-on.
- This avoids surprising users on slow or restricted networks while still allowing a genuinely automatic mode.

## Open Decision Needed

Pick the trigger scope first:

- browse-only lazy auto scrape
- scan-time missing-profile auto scrape
- a user setting that enables scan-time auto scrape
