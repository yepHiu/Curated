/**
 * Frontend comparison form for actor names and aliases.
 *
 * JavaScript does not expose Unicode Default Case Folding directly. Applying
 * upper-case expansion before lower-casing covers multi-code-point folds such
 * as German sharp-s and unifies positional forms such as Greek final sigma,
 * while NFKC and whitespace folding mirror the backend identity boundary.
 */
export function normalizeActorIdentity(value: string): string {
  return value
    .normalize("NFKC")
    .trim()
    .replace(/\s+/gu, " ")
    .toUpperCase()
    .toLowerCase()
}
