# Homepage Daily Recommendations Design

Status: Draft, pending user approval
Date: 2026-04-15

## Goal

Move homepage `Hero` and `今日推荐` from frontend-only deterministic assembly to backend-persisted UTC daily recommendation snapshots so every device and browser sees the same results. Keep the results rotating daily, enforce no overlap with the previous UTC day when inventory allows, and bias selection toward broader library coverage instead of repeatedly surfacing the same top titles.

## Confirmed Product Decisions

- Recommendation source of truth lives in the backend.
- Scope includes both homepage `Hero` and homepage `今日推荐`.
- Recommendation day boundary remains UTC.
- Cross-day behavior must update automatically even if the app stays open.
- Previous-day recommendation state must be backend-persisted so all devices and browsers stay consistent.
- “Yesterday” means the previous UTC calendar day, not the last generated snapshot day.
- When the library cannot satisfy full no-repeat coverage, quantity wins:
  fill as many slots as possible from non-yesterday titles first, then allow controlled reuse.

## Architecture

- Add a backend daily snapshot store keyed by UTC date.
- Backend owns generation, persistence, and fallback behavior for daily homepage recommendations.
- Frontend fetches the current UTC-day snapshot and uses it for `Hero` and `今日推荐`.
- Existing frontend logic continues to compute `recentMovies`, `continueWatching`, and `tasteRadar`.

## Daily Snapshot Model

Each snapshot should persist at least:

- `date_utc`
- `hero_movie_ids`
- `recommendation_movie_ids`
- `generated_at`
- `generation_version`

Behavior:

- If today’s snapshot exists, return it unchanged.
- If today’s snapshot does not exist, generate it from current library state and persist it.
- If yesterday’s snapshot exists, treat its `Hero + 今日推荐` union as the previous-day exclusion set.
- If yesterday’s snapshot does not exist, generate today normally without previous-day exclusion.

## Recommendation Algorithm

### Slot layout

- Total daily homepage recommendation footprint: `14` titles
- `Hero`: `8`
- `今日推荐`: `6`
- No same-day overlap between `Hero` and `今日推荐`

### Candidate filtering

- Exclude trashed titles from both sections.
- Continue excluding `FC2` titles from `Hero`.
- Use yesterday’s `Hero + 今日推荐` union as the default exclusion set for today.

### Selection strategy

Use a layered scoring model instead of pure randomness:

`base_score = quality + preference + freshness - exposure_penalty - recent_repeat_penalty - diversity_penalty`

Suggested dimensions:

- `quality`
  - favorite boost
  - user rating / movie rating boost
  - artwork availability boost
- `preference`
  - actor, tag, and studio affinity from ratings, favorites, and playback signals
- `freshness`
  - newly imported but under-exposed titles gain weight
- `exposure_penalty`
  - titles shown many times in a rolling window lose weight
- `recent_repeat_penalty`
  - very recent homepage appearances lose more weight than older appearances
- `diversity_penalty`
  - soft penalty when the same actor, tag, or studio already dominates today’s slate

### Diversity balancing

Avoid a homepage where one actor, one tag family, or one studio occupies too many slots.
This should be implemented as soft caps via additional score penalties, not hard rejection,
so small libraries still produce a full slate.

### Generation flow

1. Resolve `today_utc`.
2. Return today’s snapshot if already stored.
3. Load yesterday’s snapshot if it exists.
4. Build the active candidate pool.
5. Build the previous-day exclusion set from yesterday’s `Hero + 今日推荐`.
6. Select `Hero 8` from eligible candidates.
7. Select `今日推荐 6` from the remaining eligible candidates.
8. If fewer than 14 non-yesterday titles are available:
   - fill from non-yesterday titles first
   - backfill from yesterday titles only when necessary
   - prefer least-recently-exposed titles during backfill
9. Persist the finished snapshot.

## API Shape

Add a backend homepage recommendation endpoint:

- `GET /api/homepage/recommendations`

Response should include:

- `dateUtc`
- `generatedAt`
- `heroMovieIds`
- `recommendationMovieIds`
- optional `generationVersion`
- optional `source`

Frontend behavior:

- Request the endpoint on homepage entry.
- Resolve IDs against existing movie cache.
- Use backend snapshot IDs for `Hero` and `今日推荐`.
- Keep current frontend-derived sections for recent imports, continue watching, and taste radar.

## Frontend Cross-Day Refresh

Frontend should not recompute recommendations locally.
It should detect UTC date rollover and re-fetch the backend snapshot.

Recommended behavior:

- On homepage mount, fetch `GET /api/homepage/recommendations`.
- Maintain a lightweight current UTC day key in the homepage view or a dedicated composable.
- Poll or schedule a recheck periodically.
- When the UTC day changes, re-fetch and replace `Hero` and `今日推荐`.
- If the app stays open across UTC midnight, the visible homepage updates without requiring remount.

## Failure Handling

- If the endpoint fails but the page already has a snapshot loaded:
  keep showing the last successful snapshot and retry later.
- If the first homepage load fails in Web API mode:
  degrade to the existing frontend recommendation assembly so the homepage is not empty.
- Mock mode can keep using the current frontend-only algorithm and does not promise cross-device consistency.

## Testing Scope

### Backend tests

- Same UTC day returns the same snapshot after first generation.
- A new UTC day produces a new snapshot.
- When inventory allows, today’s `14` titles do not overlap with yesterday’s `14`.
- When inventory is insufficient, non-yesterday titles are exhausted before yesterday titles are reused.
- Diversity and exposure penalties reduce repetition over repeated day generations.
- Concurrent same-day requests do not create conflicting snapshots.

### Frontend tests

- Homepage uses backend snapshot IDs for `Hero` and `今日推荐`.
- Homepage re-fetches after UTC date rollover.
- Failed refresh keeps the current snapshot when one is already loaded.
- Initial endpoint failure falls back to the existing frontend assembly.

### Regression tests

- Existing hero carousel interaction and rendering remain intact.
- `recentMovies`, `continueWatching`, and `tasteRadar` behavior remain intact.

## Migration And Observability

### Migration

- Introduce a new daily homepage snapshot table without backfilling historical data.
- On the first post-release day:
  - if yesterday's snapshot does not exist, generate today's snapshot without previous-day exclusion
  - from the second generated UTC day onward, previous-day exclusion starts applying normally

### Concurrency Safety

- Enforce a unique database constraint on `date_utc`.
- Generation flow should use read-first, write-once behavior:
  - read today's snapshot
  - if absent, attempt insert
  - if another request wins the insert race, re-read and return the persisted row

### Logging And Diagnostics

- Emit one backend summary log per generation attempt including:
  - `date_utc`
  - whether an existing snapshot was reused or a new one was generated
  - active candidate pool size
  - previous-day exclusion size
  - whether fallback reuse from yesterday was required
  - selected `hero_movie_ids`
  - selected `recommendation_movie_ids`
- Persist or log `generation_version` with the snapshot so future algorithm changes remain traceable.
- Keep room for optional per-title explanation diagnostics in logs or internal debug responses, such as:
  - preference match
  - freshness boost
  - exposure penalty
  - diversity penalty

## Docs Impact

If implemented, update the following docs:

- `.cursor/rules/project-facts.mdc`
- `README.md`
- `CLAUDE.md`
- `API.md`

Update any homepage recommendation description to clarify:

- backend-owned UTC daily snapshots
- cross-device consistency requirement
- fallback behavior in mock mode
