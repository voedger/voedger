---
registered_at: 2026-04-15T12:35:45Z
change_id: 2604151235-fix-test-recovery
baseline: 74a37e0151cf28397a5fd202018745bc661b486d
issue_url: https://untill.atlassian.net/browse/AIR-3572
archived_at: 2026-04-15T13:10:07Z
---

# Change request: Fix TestRecovery

## Why

The `TestRecovery` test is failing in CI. See [issue.md](issue.md) for details.

## Root cause

Race condition in the command processor service loop (`provide.go`). The `appsPartitions` map clearing was only inside the `select`'s `<-vvmCtx.Done()` case, but the `for vvmCtx.Err() == nil` loop condition could exit the loop before reaching the `select`:

1. Service goroutine processes a command and calls `sendResponse`
2. Test goroutine receives the response, `sendCUD` returns, `restartCmdProc` calls `app.cancel()` — context is now cancelled
3. Service goroutine is still executing post-`sendResponse` code (metric updates, etc.)
4. Service goroutine reaches `for vvmCtx.Err() == nil` — condition is false, loop exits without entering the `select`
5. `appsPartitions` is never cleared because the clearing code was only reachable via the `<-vvmCtx.Done()` select case
6. On the next service run, `getAppPartition` finds old partition data, recovery is skipped, no recovery log lines are produced
7. `HasLine` assertion fails because the expected recovery log lines were never written

Secondary issue: the test's request handler callback used a captured `vvmCtx` instead of the `requestCtx` parameter when creating `NewCommandMessage`, resulting in a cancelled context being passed as `RequestCtx()` after restart

## What

- Move `appsPartitions` clearing after the `for` loop so both exit paths (loop condition and `<-vvmCtx.Done()` select case) converge on it
- Use `requestCtx` parameter instead of captured `vvmCtx` in the test's request handler callback, matching the production code pattern
