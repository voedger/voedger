---
change_id: 2611201746-actualizer-optimize-logging
type: perf
issue_url: https://untill.atlassian.net/browse/AIR-5307
domains: [prod]
scope: [apps, observability]
---

# Change request: Reduced actualizer logging

Refs:

- [AIR-5307: voedger, acutalizer: optimize logging](./issue-AIR-5307.md)

## Why

When a new projector processes the event log from FirstOffset, the actualizer emits per-event debug messages containing argument payloads and trigger details. For large logs, these messages consume excessive space in container logs and Grafana.

## What

- The actualizer emits no per-event `triggeredby` detail messages while a new projector processes historical events from FirstOffset
- A new projector emits one Info entry at `ap.entire_log_run.start` before an entire log run from FirstOffset and one at `ap.entire_log_run.complete` after processing and flushing finish; retries do not duplicate these entries
- Both entire log run lifecycle entries contain application, projector, and partition tags with an empty message and no `args` or `poffset`; completion also includes elapsed `duration_ms`
- Actualizer log messages omit the `args` payload, including during normal event processing
- Errors during an entire log run remain logged with all applicable diagnostic context tags, including application, projector, workspace, event name, and log offsets, without argument payloads
- Preserve projector execution, event processing order, flushing, persisted projections, and API responses; report completion only after pending entire log run work has been flushed

## How

Decisions:

- Identify new projectors from their initial saved-offset state and apply quiet logging while they process the entire log from FirstOffset through the existing startup catch-up phase; do not apply suppression merely because an existing projector resumes from a saved offset
- Carry the logging policy with each queued event so pipeline delays cannot cause historical events to acquire live-processing logging behavior
- Keep one stored entire log run lifecycle state; derive the entire log run flag per read from its starting offset instead of maintaining a separate mutable suppression field
- Suppress actualizer-generated per-event verbose details during a new projector's entire log run
- Enrich the event log context independently of verbose message emission and log level, and preserve that context for invocation and error handling even when entire log run detail logging is suppressed
- Keep argument serialization optional at the shared event-logging boundary, disabling it for asynchronous actualizers while preserving command-processing and partition-recovery logging
- Keep entire log run progress in the actualizer with `start()` and `complete()` methods; use `waitForEntireLogRunCompletion()` to enqueue an end-of-log marker and wait for its one-shot completion signal after successful flushing before reporting completion and watching live notifications, retain the start time across retries, and omit completion when processing fails or is canceled
- Inherit the actualizer context for lifecycle logging and add `duration_ms` only in `complete()` and use the existing `logger.InfoCtx` without message arguments; preserve the logger package and its standard `msg=""` format
- Verify the policy through actualizer execution and captured logs, checking projection progress alongside entire log run suppression, the transition to live processing, existing-projector resumption, and error context tags without arguments at verbose and non-verbose log levels

Assumptions:

- The request to run silently concerns routine actualizer-generated event logs; error diagnostics and logs explicitly emitted by projector extensions remain available

Out of scope:

- Changing synchronous projector logging or command argument logging
- Changing the logger package, global log levels, log storage configuration, or retention policies

References:

- [Asynchronous event reading, queuing, logging, and retry lifecycle](../../../../../pkg/processors/actualizers/async.go)
- [Shared event and CUD logging](../../../../../pkg/processors/utils.go)
- [Command-processing and partition-recovery logging callers](../../../../../pkg/processors/command/impl.go)
- [Actualizer logging, recovery, and notification tests](../../../../../pkg/processors/actualizers/async_test.go)
- [Synchronous projector logging](../../../../../pkg/processors/actualizers/impl.go)

## Functional design

- [x] create: [apps/logging.feature](../../../../specs/prod/apps/logging.feature)
  - Feature specification with three short logging scenarios for a completed entire log run, pending processing, and automatic retry

## Technical design

- [x] update: [Application logging technical design](../../../../specs/prod/apps/logging--td.md)
  - update: asynchronous actualizer logging to suppress per-event verbose details during an entire log run from FirstOffset for a new projector, distinguishing this phase from live processing and existing-projector resumption
  - update: asynchronous actualizer event messages to omit argument payloads while preserving command-processing and partition-recovery logging
  - update: event context enrichment and propagation so error logging retains all applicable tags, including `vapp`, `extension`, `wsid`, `woffset`, `poffset`, and `evqname`, independently of verbose logging and entire log run suppression, without `args`
  - add: per-event logging policy propagation through the asynchronous pipeline and verification requirements for an entire log run, live processing, resumption, and error/retry diagnostics
  - add: entire log run start and completion entries using lifecycle contexts separate from event contexts and empty messages, completion after flushing, and lifecycle behavior for retry, empty logs, and cancellation

## Construction

- [x] update: [pkg/processors/utils_test.go](../../../../../pkg/processors/utils_test.go)
  - add: tests for event context enrichment at Verbose and Info levels, asserting inherited and event-specific tags through emitted error logs without argument access or detail logging
  - add: tests that argument logging can be disabled without reading `ArgumentObject()`, while trigger messages contain no `args` placeholder or leading separator and CUD filtering remains effective
  - add: compatibility coverage for the existing argument-logging default and the detail logger's non-verbose early return
  - reuse: existing coreutils event and row test doubles, exercising production logging helpers and their observable output

- [x] update: [processors/actualizers/async_test.go](../../../../../pkg/processors/actualizers/async_test.go)
  - update: existing logging tests to exercise normal processing outside an entire log run, retaining function-trigger and record-trigger CUD filtering coverage without argument payloads
  - add: execution tests for the entire log run, live-processing, saved-offset resumption, delayed pipeline, and empty initial log cases in the [logging verification matrix](../../../../specs/prod/apps/logging--td.md#async-projectors)
  - add: event-error coverage at Verbose and Info levels, including failures before invocation and detail-logging failures, asserting the error and all applicable tags in the same entry without `args`
  - update: retry coverage to verify successful processing and logging policy when initialization resumes from either NullOffset or a saved nonzero offset
  - add: a failure during an entire log run followed by saved-offset retry, verifying normal retry logging with one runtime start and completion
  - add: an entire log run spanning multiple read batches, verifying that later batches retain quiet logging as the reader offset advances
  - Verify: lifecycle entries include application, projector, and partition tags with empty messages and no `args`, `poffset`, or event-specific tags; only completion includes duration and it follows flushing
  - Adapt: the three scenarios in `logging.feature` into explicit `logging: scn:` subtests in `TestAsyncLogging`, with verbatim step comments beside the setup, actions, and assertions they describe
  - Verify: pending-processing assertions inspect one captured log snapshot; automatic retry checks once-only start and completion in order around the error at Verbose and Info levels
  - Retain: empty-log, cancellation, skipped-event, multi-batch, saved-offset, and live-event regression coverage without unrelated Gherkin scenario labels or step comments
  - verify: invocation or persisted projection results and offset progress before asserting missing per-event logs during an entire log run; use bounded synchronization and independent subtests without parallel mutation of global logging state

- [x] update: [processors/command/impl_test.go](../../../../../pkg/processors/command/impl_test.go)
  - add: a successful command with a nonempty argument payload and assert that `cp.plog_saved` still contains its serialized arguments and event tags
  - retain: CUD logging and non-verbose behavior coverage to detect changes leaking from the asynchronous actualizer policy

- [x] update: [processors/command/recovery_checkpoint_test.go](../../../../../pkg/processors/command/recovery_checkpoint_test.go)
  - add: a partition-recovery case that reaches last-event reapplication with nonempty arguments and verifies the `cp.partition_recovery.reapply` entry retains its payload and event tags
  - reuse: the existing recovery setup and completion helpers, asserting successful recovery as well as the captured log entry

- [x] update: [pkg/processors/utils.go](../../../../../pkg/processors/utils.go)
  - extract: reusable event context enrichment independent of log level, argument access, and message emission
  - add: an additive entry point for caller-selected argument logging, sharing the formatter and CUD handling while preserving the existing exported `LogEventAndCUDs` signature and default behavior
  - implement: argument-disabled formatting and serialization avoidance as defined in the [shared logging design](../../../../specs/prod/apps/logging--td.md#logeventandcuds), preserving CUD callbacks and error propagation

- [x] update: [internal/actualizers/actualizers.go](../../../../../pkg/appparts/internal/actualizers/actualizers.go)
  - Add: the partition ID to the context created by `PartitionActualizers.startNews()` so lifecycle and event logs inherit `partid`

- [x] update: [processors/actualizers/async.go](../../../../../pkg/processors/actualizers/async.go)
  - capture: `entireLogRun` once per startup read before offsets advance, pass it through all batches, and attach it to each queued workpiece without a separate mutable actualizer flag
  - apply: suppression only to startup reads from FirstOffset; notification-driven reads and saved-offset resumption use normal logging, with policy recalculated on retry
  - Log: one Info start before the first suppressed startup read and one completion after queued entire log run processing and flushing, retaining lifecycle state across retries and using lifecycle contexts separate from event contexts without message arguments
  - enrich: the workpiece context before partition borrowing and workspace checks, and retain it for invocation and error reporting even when detail logging is skipped or fails
  - omit: argument logging for asynchronous actualizers and bypass event, CUD, and success detail emission for suppressed workpieces according to the [async projector design](../../../../specs/prod/apps/logging--td.md#async-projectors)
  - preserve: event and partition release, intent application, flushing, offset persistence, retry, cancellation, and error metrics; signal completion through an end-of-log marker only for pending entire log runs, using a fresh channel on retry and clearing the signal after a successful flush

- [x] verify: format and validate the affected packages
  - run: `gofmt` on changed Go files and `git diff --check`
  - run: `go test ./pkg/goutils/logger ./pkg/processors ./pkg/processors/actualizers ./pkg/processors/command`
  - run: `go test -race ./pkg/processors/actualizers` to check reader/pipeline lifecycle propagation and test synchronization
  - review: the cumulative diff for logging scope, error-context retention, resource lifecycle, and test assertions; report any checks blocked by the environment
