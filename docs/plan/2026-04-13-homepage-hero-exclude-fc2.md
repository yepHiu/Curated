# Homepage Hero Exclude FC2 Plan

## Goal

Hero daily picks must never show FC2 movies, and Hero must still render 8 posters by repeating non-FC2 titles when the non-FC2 pool is smaller than 8.

## Scope

- Only change the homepage Hero candidate pool.
- Keep FC2 movies visible in library, search, details, recent imports, continue watching, and preference recommendations unless another rule filters them.
- Detect FC2 through the normalized movie `code`, matching codes such as `FC2-123456`, `fc2-123456`, `FC2PPV-123456`, and `FC2 PPV 123456`.

## Implementation

1. Add a unit test in `src/lib/homepage-portal.test.ts`.
   - Build a movie list with fewer than 8 non-FC2 movies plus multiple FC2 variants.
   - Assert every `heroMovies` item is non-FC2.
   - Assert Hero is still filled to 8 items by repeating non-FC2 movies.

2. Update `src/lib/homepage-portal.ts`.
   - Add a small helper that recognizes FC2 movie codes.
   - Use that helper only for `heroMovies` selection.
   - When the filtered Hero pool has at least 1 movie but fewer than `heroLimit`, repeat the seeded non-FC2 order until the requested size is reached.
   - Keep the rest of `activeMovies` unchanged for other homepage sections.

3. Verify.
   - Run `pnpm vitest run src/lib/homepage-portal.test.ts --configLoader native --pool threads`.
   - If needed, run the homepage view tests afterward because the visible hero rail assumes 8 items in some fixtures.
