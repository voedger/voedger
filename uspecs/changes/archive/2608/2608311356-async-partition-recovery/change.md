---
change_id: 2608311046-async-partition-recovery
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-4571
domains: [prod]
scope: [apps]
breaking: true
---

# Change request: Asynchronous command partition recovery

## Why

Command partition recovery scans the complete partition PLog and can take progressively longer as event volume grows. Running that work synchronously in the command processor delays commands for unrelated partitions and reduces platform responsiveness during recovery.

## What

Symptom: Recovering one command partition blocks command handling for every partition assigned to the same command processor until recovery completes.

```text
Client submits a command
      |
      v
Command processor service loop dequeues the request
      |
      v
getAppPartition finds no cached application partition
      |
      v
cmdProc.recovery scans the partition PLog synchronously
      |               <-- fault: recovery runs inline in the command-processing loop
      v
Commands for recovering and unrelated partitions wait   (symptom)
```

Corrected behavior: The command processor runs recovery in a separately tracked goroutine per application partition, prevents duplicate recovery for the same partition, and continues handling commands for partitions that are not recovering.

## How

Decisions:

- Keep recovery demand-driven and key command-processor state by application QName and partition ID; reset only the affected partition state after a storage or sync-actualizer failure so the next request starts fresh recovery.
- Encapsulate partition lifecycle in a dedicated partition-recovery manager backed by one keyed map whose entries contain recovered partition data, the recovering marker, and the retained recovery error; keep synchronization, worker tracking, test hooks, state reset, and shutdown in the same manager.
- Represent partition lifecycle explicitly as recovering, failed, or ready behind synchronized state transitions, and publish recovered offsets, workspace state, and ID generators only after the entire recovery succeeds.
- Return a retryable `503 Service Unavailable` for the command that starts recovery and for later commands targeting that partition while recovery is in progress; do not retain request messages for replay.
- Retain a recovery failure until the next command for that partition; that command starts a fresh recovery attempt and returns `500 Internal Server Error` with the prior recovery error. Commands received while the restarted recovery is running receive `503 Service Unavailable`.
- Give each recovery worker its own borrowed application-partition resources and recovery workpiece so it does not retain or race with the initiating request's workpiece lifecycle.
- Associate each recovery worker with the exact partition-state instance that started it, and discard its result if that state was reset or replaced before the worker completed.
- Derive recovery lifetime from the command-processor service context rather than a request context, cancel workers during service shutdown, and wait for all started workers before clearing processor state or closing shared pipelines.
- On recovery failure, log through the existing partition-recovery stages, discard partial state, clear the recovering marker, and retain the error for the next command that initiates a fresh attempt.
- Preserve serialized command execution and the current PLog scan/re-apply algorithm; concurrency is introduced only between recovery workers and command handling for other partitions.
- Verify concurrency and shutdown behavior with deterministic recovery gates rather than timing sleeps.

Assumptions:

- Command callers treat `503 Service Unavailable` as retryable and do not require transparent server-side queuing while their partition recovers.
- Recovery of different application partitions can safely proceed concurrently when each worker owns its borrowed partition resources.

Out of scope:

- Optimizing or replacing the full PLog recovery scan and sequence reconstruction algorithm.
- Buffering and replaying commands received while their target partition is recovering.
- Parallelizing ordinary command execution after partition recovery.

References:

- [command processor service and lifecycle](../../../../../pkg/processors/command/provide.go)
- [partition lookup and recovery flow](../../../../../pkg/processors/command/impl.go)
- [existing recovery behavior](../../../../../pkg/processors/command/impl_test.go)
- [application processing architecture](../../../../../uspecs/specs/prod/apps/arch-processing.md)
- [known recovery scaling constraint](../../../../../uspecs/specs/prod/apps/arch2-sequences.md)
- [partition recovery logging contract](../../../../../uspecs/specs/prod/apps/logging--td.md)

## Construction

- [x] update: [command/impl_test.go](../../../../../pkg/processors/command/impl_test.go)
  - update: the command test fixture to deploy and route multiple partitions and use channel-controlled recovery gates and start counters without timing sleeps
  - update: existing recovery cases for the asynchronous `503 Service Unavailable` then retry flow while preserving offset, record-ID, last-event re-apply, and structured-log assertions
  - add: deterministic coverage that blocks one partition's recovery and proves a command for another partition completes without waiting
  - add: coverage that concurrent commands for one recovering partition start exactly one worker and receive `503 Service Unavailable` until recovery publishes ready state
  - add: recovery-failure coverage proving partial state is not published and the next command returns `500 Internal Server Error` with the failure while starting a successful retry
  - add: service-shutdown coverage proving active recovery is cancelled and joined without sleeps or leaked borrowed resources
  - add: deterministic coverage proving an old recovery worker cannot restore reset state or overwrite a replacement recovery state

- [x] update: [sys/it/impl_bootstrap_test.go](../../../../../pkg/sys/it/impl_bootstrap_test.go)
  - update: bootstrap integration calls to pass the retry-enabled federation view required by the bootstrap contract

- [x] update: [command/provide.go](../../../../../pkg/processors/command/provide.go)
  - update: the service factory to initialize the dedicated partition manager and inject deterministic recovery test hooks through a private factory seam
  - update: the command pipeline to resolve partition readiness after authentication and workspace validation, and before rate limiting, command lookup, and authorization
  - update: command-service shutdown to cancel and join recovery workers before shared pipelines are closed and processor state is cleared
  - update: partition restart scheduling to reset only the affected application-partition state instead of every partition of the application

- [x] update: [command/impl.go](../../../../../pkg/processors/command/impl.go)
  - update: `getAppPartition` to atomically reuse ready state, deduplicate recovery startup, and return retryable `503 Service Unavailable` while the target partition is recovering
  - update: recovery startup to use `toRecoveryWorkpiece` to transfer the borrowed application-partition resources into an independently owned workpiece tied to the service context
  - update: recovery completion to publish fully reconstructed partition state atomically on success, or discard partial state and retain the failure until the next request starts a retry
  - update: recovery completion to discard results from workers whose original partition state was reset or replaced while they were running
  - update: recovery resource cleanup to release borrowed partition resources and the last PLog event on cancellation and error exits
  - preserve: the current full-PLog scan, sequence reconstruction, last-event re-apply, and partition-recovery logging stages

- [x] add: [command/test_utils.go](../../../../../pkg/processors/command/test_utils.go)
  - add: deterministic partition-recovery hooks and test control for worker start counts, blocking gates, injected failures, completion waiting, and cancellation
  - add: a request-sender wrapper that waits for test-controlled recovery after `503 Service Unavailable` while leaving raw requests available for status assertions

- [x] update: [command/types.go](../../../../../pkg/processors/command/types.go)
  - add: the application-partition key and dedicated partition-manager types, including service-scoped worker tracking and injected test hooks
  - add: one unified keyed partition-state map whose entries embed recovered partition data and carry the recovering marker and retained recovery error
  - add: the recovery function contract used by the partition manager to invoke the existing recovery algorithm

- [x] update: [federation/interface.go](../../../../../pkg/coreutils/federation/interface.go)
  - document: that plain federation calls do not retry automatically and that `WithRetry` applies the default retryable statuses, retry windows, `Retry-After` handling, and jittered backoff policy

- [x] update: [btstrp/impl.go](../../../../../pkg/btstrp/impl.go)
  - update: bootstrap deployment to require the retry-enabled federation interface so transient partition-recovery responses are retried

- [x] update: [vvm/provide.go](../../../../../pkg/vvm/provide.go)
  - update: bootstrap operator wiring to pass `federation.WithRetry()` into bootstrap deployment

- [x] update: [vvm/wire_gen.go](../../../../../pkg/vvm/wire_gen.go)
  - regenerate: dependency-injection wiring for the retry-enabled bootstrap federation argument
