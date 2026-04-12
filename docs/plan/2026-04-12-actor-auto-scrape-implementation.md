# Actor Auto Scrape Implementation Plan

Date: 2026-04-12

## Goal

Add a persisted setting `autoActorProfileScrape` so Curated can automatically enqueue missing actor profile scrapes during the movie metadata scrape pipeline.

## Scope

- Add backend config/settings persistence for `autoActorProfileScrape`
- Expose the field through `GET /api/settings` and `PATCH /api/settings`
- When enabled, after movie metadata is saved, enqueue `scrape.actor` only for actors that still have neither avatar nor summary
- Keep existing `ActorProfileCard` lazy auto-scrape behavior unchanged
- Add a Settings page toggle in Web API and Mock modes

## Non-Goals

- No always-on background actor scraping without user opt-in
- No scheduled actor scrape maintenance job
- No change to the current actor-profile read endpoint semantics

## Implementation Notes

1. Backend setting and persistence
   - Add `AutoActorProfileScrape` to backend config and library settings merge logic
   - Mirror the existing persisted toggle pattern used by `autoLibraryWatch`

2. Settings contract
   - Extend backend and frontend settings DTO/patch types
   - Add runtime controller methods on `App`

3. Auto enqueue timing
   - Trigger from the movie metadata scrape pipeline after `SaveMovieMetadata`
   - Use scraped actor names from the movie metadata result, not raw scan discovery
   - Skip actors whose profile already has avatar or summary
   - Dedupe within the current process so the same actor is not auto-enqueued repeatedly while one auto scrape is already pending

4. UI
   - Add a toggle near the existing auto-scan / auto-scrape behavior settings
   - Default remains off to keep outbound scraping conservative

5. Verification
   - Backend tests for settings merge and settings API
   - Backend tests for auto-enqueue behavior
   - Frontend typecheck
