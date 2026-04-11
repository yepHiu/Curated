# Dev Performance Monitor Hidden Entry Adjustment

## Confirmed Change

- Hidden state no longer keeps a separate bottom-edge `Show Perf` button.
- The restore entry is attached to the existing dev-environment watermark in `AppShell`.
- The `dev` watermark remains a non-interactive environment marker.
- When the performance monitor is hidden, a compact `perf` button appears beside the `dev` badge and restores the monitor on click.
- Visibility preference remains local-only and continues to use `localStorage`.

## Implementation Notes

- `DevPerformanceBar` owns hide actions but no longer renders the restore entry.
- `AppShell` renders the dev-environment watermark group and decides whether to show the compact restore button.
- Hidden-state preference is shared through a small dev-performance visibility helper so watermark and monitor stay in sync.
