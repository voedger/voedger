---
change_id: 2611190722-rejected-command-test
type: test
issue_url: https://untill.atlassian.net/browse/AIR-5301
domains: [prod]
scope: [apps]
---

# Change request: Rejected command regression test correction

Refs:

- [AIR-5301: fix TestRejectedCommandsDoNotRecoverOrWrite](./issue-AIR-5301.md)

## Why

The CI race detector reports concurrent access to the logger's context writers in `TestRejectedCommandsDoNotRecoverOrWrite`: `logger.StartCapture` calls `SetCtxWriters` while the notification broker heartbeat reads those writers through `logCtx`. This test-lifecycle race fails command processor regression coverage that protects rejected commands from triggering partition or Workspace recovery or appending events to their logs.

## What

- Log capture setup and restoration do not race with background logging during the command processor test lifecycle
- Command processor regression coverage reliably verifies repeated command rejection when the Application partition cannot be borrowed
- Rejection assertions verify the response status and error message required by the command processor contract
- Rejected commands leave PLog and WLog offsets unchanged and start neither partition nor Workspace recovery
- Rejected commands produce no partition-recovery, Workspace-recovery, or PLog-saved log entries attributable to those commands
- The correction preserves production behavior and exercises the real command processor rather than asserting only configured test-double results

## How

Decisions:

- Make log capture encompass the test fixture's lifetime, installing capture before background services start and restoring logger state only after those services have stopped
- Keep log assertions focused on rejected-command execution by clearing captured setup output through the existing synchronized capture buffer rather than replacing global writers while services run
- Retain the existing fixture and teardown mechanisms and use repeated race-enabled execution of the affected test to verify the lifecycle correction

Assumptions:

- No unrelated test concurrently reconfigures the global logger or leaves background log producers running across test boundaries

Out of scope:

- General synchronization changes to the logger's global configuration
- Refactoring other tests that use log capture

References:

- [Command test setup and service teardown](../../../../../pkg/processors/command/impl_test.go)
- [Capture restoration and synchronized buffer reset](../../../../../pkg/goutils/logger/logcapture.go)
- [Context writer replacement and logging reads](../../../../../pkg/goutils/logger/loggerctx.go)
- [Notification broker startup and shutdown waiting](../../../../../pkg/in10nmem/provide.go)

## Construction

- [x] Update: [processors/command/impl_test.go](../../../../../pkg/processors/command/impl_test.go)
  - Initialize a test-scoped assertion instance at the start of `TestRejectedCommandsDoNotRecoverOrWrite` and use it throughout that test
  - Start log capture before `setUpRecoveryTestApp` launches the notification broker and command processor
  - Preserve deferred fixture teardown so background services stop before the capture's registered cleanup restores global logger state, including on assertion failure
  - Reset the synchronized capture buffer after setup and baseline log reads, immediately before submitting rejected commands
  - Preserve the three rejection responses, error-message assertions, unchanged PLog and WLog offsets, zero recovery-start counts, and absence of recovery or PLog-saved log entries
  - Run `gofmt` on the modified test file and `go test -race ./pkg/processors/command -run '^TestRejectedCommandsDoNotRecoverOrWrite$' -count=20`; investigate any remaining lifecycle race without widening the change to unrelated tests
