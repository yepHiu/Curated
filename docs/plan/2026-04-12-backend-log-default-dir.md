# Backend Log Default Directory

## Goal

Adjust backend log path semantics so clearing `logDir` no longer disables file logging.

## Rule

- Release builds:
  Empty `logDir` resolves to `LOCALAPPDATA\Curated\logs`.
- Development builds:
  Empty `logDir` resolves to `backend/runtime/logs`.
- Explicit non-empty `logDir` still wins.

## Scope

- Backend config defaulting and settings patch behavior
- Settings-page copy describing empty-directory behavior
- Repository memory/docs that previously said empty `logDir` means console-only

## Implementation Notes

- Added build-specific `defaultLogDir()` helpers in `backend/internal/config`.
- Normalized effective log directory through `config.ResolveLogDir(...)`.
- Updated backend settings responses so the UI sees the effective directory after clearing the field.
- Added tests for dev defaults, release defaults, and `SetBackendLogPatch` fallback behavior.
