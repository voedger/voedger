---
registered_at: 2026-03-18T08:22:25Z
change_id: 2603180822-validators-return-error
issue_url: https://untill.atlassian.net/browse/AIR-3339
baseline: aeff999f2fc662c2a5f2679fc90d803ab16c4d95
---

# Change request: Router validators return error instead of reply

## Why

Individual validator functions (`cookiesTokenToHeaders`, `readBody`) currently write HTTP responses and log errors directly, mixing transport concerns into reusable validation logic. Centralizing error reply and logging in `validate()` improves separation of concerns and makes validators easier to test and compose.

## What

Refactor validator functions in `pkg/router/impl_validation.go` so that error handling is centralized:

- `cookiesTokenToHeaders()` and `readBody()` return an error instead of writing HTTP responses or logging directly
- `validate()` becomes the single place that replies with HTTP 400 Bad Request and logs errors when a validator returns an error
