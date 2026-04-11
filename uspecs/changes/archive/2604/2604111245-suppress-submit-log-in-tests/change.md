---
registered_at: 2026-04-11T12:08:16Z
change_id: 2604111208-suppress-submit-log-in-tests
baseline: 8d476f5157b295460b555fd5068ba5ffd4aaf1c6
issue_url: https://untill.atlassian.net/browse/AIR-3551
archived_at: 2026-04-11T12:45:28Z
---

# Change request: Suppress processor submit failure logging in tests

## Why

During stress testing the console is overflowed with "no processors available" error messages from `replyCommandBusy` and `replyQueryBusy`. These log lines are expected under load in tests and produce noise that obscures real failures.

See [issue.md](issue.md) for details.

## What

Suppress error logging on processor submit failure when running in test mode:

- Add a mechanism to detect test mode without using the `testing` package or command-line args inspection
- Skip the `logger.ErrorCtx` calls in `replyCommandBusy` and `replyQueryBusy` when test mode is active
