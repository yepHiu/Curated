# Homepage Recommendation Repeat Analysis

## Current Behavior

The homepage recommendation backend currently persists one snapshot per UTC day and reuses that same snapshot for the rest of the UTC day.

Observed recommendation generation behavior:

- Same-day duplicate prevention is already present:
  - hero and recommendation selections share the same `selected` set, so a title cannot appear in both rows on the same day.
- Cross-day duplicate prevention is only partially strict:
  - the combined hero + recommendation set from **yesterday only** is hard-excluded first;
  - older snapshots are not hard-excluded, they only contribute an exposure penalty;
  - if inventory is insufficient, yesterday titles are backfilled as the final fallback.

## Why Repeats Still Happen After 2-3 Days

The current implementation applies a soft score penalty for prior exposure over a 14-day lookback window, but that penalty is not strong enough to guarantee weekly freshness.

Concretely:

- ranking starts from `movie.Rating * 10`
- favorites get `+24`
- cover/thumb availability gets `+6`
- recent additions get up to `+10`
- past exposures only subtract a bounded penalty

That means a strong title can still re-enter the slate after the "yesterday" hard block expires, especially when:

- it has a high rating,
- it is favorited,
- it has strong metadata completeness,
- or the library has strong score concentration near the top.

So the current algorithm is best described as:

- "avoid immediate repeats from yesterday"
- "discourage recent repeats"
- not "guarantee no repeats within a week"

## Root Cause Summary

This is not mainly a hero/recommendation same-day de-duplication bug.

It is a policy gap:

- **Hard exclusion window:** only 1 day
- **Soft penalty window:** 14 days
- **Desired user expectation:** roughly 7-day freshness when inventory allows

Those are mismatched.

## Recommended Direction

Keep daily snapshots, but change cross-day freshness from "softly discourage" to "hard avoid for a rolling 7-day window, with graceful fallback".

Recommended selection strategy:

1. Build a combined set of movie IDs shown in the last 7 days.
2. First pass: hard-exclude all titles from that 7-day set.
3. If not enough titles remain, degrade gradually:
   - retry with 5-day exclusion
   - then 3-day exclusion
   - then 1-day exclusion
   - finally allow anything
4. Keep existing diversity penalties and same-day dedupe.

This matches the stated product goal:

- if the library is large enough, users should not see repeats within the week;
- if the library is too small, the system still returns a complete slate instead of failing.

## Alternative: Refresh Only On Monday And Thursday

This is technically possible, but it solves a different problem.

What it improves:

- fewer regeneration cycles
- less visible day-to-day churn
- easier mental model for users

What it worsens:

- the same hero/recommendation slate would remain visible for 3-4 days at a time
- new library additions feel slower to surface
- stale recommendations become more noticeable during a long interval

So if the goal is "do not see the same movies again within a week", a Monday/Thursday refresh cadence is not the best primary fix. It trades repetition frequency for longer snapshot persistence.

## Implemented Policy

The backend policy is now:

- daily homepage snapshots remain the default cadence;
- same-day hero/recommendation dedupe remains unchanged;
- cross-day hard exclusion now uses a fallback ladder:
  - `7` days
  - `5` days
  - `3` days
  - `1` day
  - `0` days
- if the library has enough inventory, the generator will avoid the entire last-7-day combined slate;
- if not, it gradually relaxes the exclusion window until the slate can be filled;
- the older exposure-penalty logic still remains, so even after fallback the ranking still discourages recently surfaced titles.

This gives the desired product behavior:

- strong freshness when inventory is sufficient;
- graceful degradation instead of empty or undersized rows when inventory is limited.

## Suggested Product Choice

Preferred default:

- daily snapshot generation
- 7-day hard exclusion with fallback ladder

Optional future setting:

- recommendation cadence:
  - daily
  - twice weekly (Monday/Thursday)

That cadence toggle should be treated as a product preference, not the main fix for repetition.
