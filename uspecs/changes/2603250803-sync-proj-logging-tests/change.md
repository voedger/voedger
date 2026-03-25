---
registered_at: 2026-03-25T08:03:36Z
change_id: 2603250803-sync-proj-logging-tests
baseline: 069c20ac5cf201476b99c576d67e9380cddc7752
issue_url: https://untill.atlassian.net/browse/AIR-3394
---

# Change request: Add logging coverage tests for sync projectors

## Why

Sync projectors perform critical operations but currently lack tests that verify all required log statements are emitted. Without such tests, logging regressions may go undetected, making production debugging and observability harder.

See [AIR-3394](https://untill.atlassian.net/browse/AIR-3394) for details.

## What

Add unit tests for sync projectors that verify all necessary logging is in place:

- Test that expected log entries (level, message, fields) are emitted during normal projector execution
- Test that error conditions produce the correct log output
- Cover edge cases where logging behavior may differ (e.g., skipped events, retries)
