---
name: curated-security-boundary-review
description: Review Curated changes at authentication, LAN/CORS, filesystem, upload, FFmpeg, or Electron preload boundaries for concrete security and privacy risks.
---

# Curated Security Boundary Review

Use this Skill for a focused security review or when a change crosses a trust boundary. It is not a replacement for normal feature implementation.

## Identify the boundary and asset

Read the changed code, closest existing tests, and relevant facts/rules. State the actor, trusted inputs, protected asset, enforced boundary, and plausible failure mode. Focus on the actual changed path rather than a generic vulnerability inventory.

## Curated boundary checks

- **Auth and sessions:** protect new API routes when PIN is enabled; do not expose bearer tokens, PINs, or secrets in DTOs, logs, errors, browser storage, or UI.
- **LAN and CORS:** preserve loopback-by-default behavior, explicit LAN/PIN requirements, exact allowed origins, host validation, and credential rules. Never reflect arbitrary origins or accept wildcard credentialed CORS.
- **Filesystem and media:** canonicalize/validate paths against configured library roots before reads, uploads, FFmpeg calls, export, deletion, or cleanup. Treat symlinks, traversal, file type, and final movie files as deliberate policy decisions.
- **Upload and external tools:** bound size, duration, concurrency, and user-controlled options; do not concatenate untrusted values into shell commands. Preserve task cancellation/error behavior.
- **Electron:** keep preload exposure narrow, validate IPC inputs in the main process, and avoid renderer access to Node/filesystem capabilities. Desktop markers must not relax API authorization.

Use stable user-safe error codes while retaining diagnostic context in structured logs without secret values. Add regression coverage for the vulnerable rejection path and legitimate allowed path. Report risks found, mitigations, tests, and any unresolved threat requiring a product decision.
