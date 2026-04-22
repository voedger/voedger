---
registered_at: 2026-03-24T09:42:13Z
change_id: 2603240942-mockable-legacy-logger
baseline: 0b9c179d97fab367cfab9cffdf91df7791c62103
issue_url: https://untill.atlassian.net/browse/AIR-3387
archived_at: 2026-03-25T09:22:35Z
---

# Change request: Mockable legacy logger functions

## Why

Legacy package-level logger functions (`logger.Verbose()`, `logger.Error()`, `logger.Info()`, etc.) write directly to stdout/stderr with no interception point, making it impossible to capture and assert log output in tests.

## What

Make legacy logger functions mockable so log output can be captured and verified in tests:

- Add `legacyOut`/`legacyErr` `io.Writer` vars to `logger.go`; update `DefaultPrintLine` to write to them instead of `os.Stdout`/`os.Stderr` directly
- Extend `StartCapture` in `logcapture.go` to redirect `legacyOut`/`legacyErr` to the in-memory captor alongside the existing slog writers, restoring originals in `t.Cleanup`
- Overhaul the test suite: remove demonstration-only tests and the `captureCtxOutput` helper; add `TestLegacyFuncs_BasicUsage`, `TestSlogFuncs_BasicUsage`, and `TestLegacyFunctions` (table-driven); convert all remaining tests to use `StartCapture`
