# Curated Frame Context Menu Implementation Plan

**Goal**

Add a right-click shortcut menu for curated-frame cards so users can delete a single frame or export that single frame as WebP/PNG without entering batch mode.

**Scope**

- Frontend only.
- Reuse the existing single-frame export flow and existing delete confirmation flow.
- Keep the interaction local to the curated frames page; no backend or storage contract changes.

**Implementation Notes**

- Create a focused `CuratedFrameContextMenu` component instead of expanding `CuratedFramesLibrary.vue` further.
- Store only the menu anchor position, selected frame, and optional actor-section context needed to preserve current single-frame export behavior.
- Disable export items when Web API is unavailable, matching the existing batch export constraint.

**Testing**

- Add a component test for the new context menu:
  - renders menu actions
  - emits export/delete actions
  - disables export actions when Web API is unavailable
- Run the targeted Vitest file first as a failing test, then again after implementation.
- Run a broader frontend verification pass before closing the task.
