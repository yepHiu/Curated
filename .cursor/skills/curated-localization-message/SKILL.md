---
name: curated-localization-message
description: Add or revise Curated user-facing copy, translations, errors, toasts, or notification-center messages across locale files and the message policy ledger.
---

# Curated Localization and Messages

Use this Skill for user-visible strings and message behavior, not for purely internal logs.

## Localize consistently

Read `src/i18n/index.ts`, the relevant keys in `src/locales/en.json`, `zh-CN.json`, and `ja.json`, plus nearby usage. Add equivalent keys in all three locales; preserve interpolation names, HTML/formatting expectations, and namespace conventions. Do not hard-code new visible copy in a component when an existing localization pattern applies.

Use stable backend error codes at the contract boundary; map them to localized user copy in the frontend rather than exposing raw server/provider errors.

## Classify messages before coding

For toast/notification-center changes, read `docs/prd/message-catalog.md` and `docs/prd/message-catalog.csv` first. Allocate a stable `MSG-xxxx` before implementation, choose the level before toast/center/badge fields, add the same ID to `src/lib/message-policy.ts`, and pass the ID from callers. Settings operations default to local/silent feedback unless product intent justifies escalation.

Run locale-related tests and, for message changes, `python3 scripts/prd/message_catalog_lint.py docs/prd/message-catalog.csv` plus the focused message-policy test. State keys and message IDs added or changed.
