---
change_id: 2611262035-release-log-events
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-5377
domains: [prod]
scope: [storage, apps, extensions]
---

# Change request: Release of log events after their last use

Refs:

- [AIR-5377: voedger: I*LogEvent.Release() missing in few places](./issue-AIR-5377.md)

## Why

A single SQL query can scan an entire partition or workspace log. Skipping cleanup on every row prevents buffer reuse and can increase allocation and garbage collection costs as the scan progresses.

## What

Symptom: Affected PLog and WLog reads leave buffers out of the pool or retain a reader reference on a cached PLog event after its last use.

```text
Query, record lookup, corruption update, or async actualizer reads the log
  |
  v
ReadPLog or ReadWLog acquires an event reference or pooled storage buffer
  |
  v
One of the following read paths ends
  |-- SQL rendering or GetORec finishes              <-- fault: callback omits Release
  |-- updateCorrupted copies the source event bytes  <-- fault: source reference is dropped
  |-- a WLog state value reaches its last use        <-- fault: no owner releases its event
  |-- an actualizer send fails or is canceled        <-- fault: unsent batch entries are abandoned
  |-- single-event decoding fails before delivery    <-- fault: reader skips cleanup
  v
Buffers remain unavailable for reuse or a PLog cache reference stays unbalanced
```

Corrected behavior: Each owner releases its event reference exactly once after its last use, including after callback errors and for unsent events on failure or cancellation, while the reader returns any resources it acquired if decoding fails before delivery.

## How

Decisions:

- Transfer one reader reference when a log callback is entered, including on PLog cache hits. The callback owner handles cleanup even if it returns an error. The cache keeps its own reference until eviction
- Keep decoding resources with the decoder until callback entry. Give each partially decoded child or CUD row an owner as soon as it acquires a buffer so a failure can unwind the unfinished row as well as the completed part of the event
- A Go caller reading `sys.WLog` releases the event after its last use of the state value and its children. A range reader releases each event in its callback or transfers the reference to the component that will finish using it
- Use successful enqueue as the ownership transfer for both actualizer read modes. Clear each transferred batch slot so cleanup on exit releases only events the reader still owns

Assumptions:

- None

Out of scope:

- Lazy event decoding and changes to the persisted log format

References:

- [Log event and callback contracts](../../../../../pkg/istructs/events-types.go)
- [Cache references on lookup and eviction](../../../../../pkg/objcache/impl.go)
- [Construction of nested event objects and CUD rows](../../../../../pkg/istructsmem/event-dynobuf.go)
- [Event and nested views exposed by WLog storage](../../../../../pkg/sys/storages/impl_wlog_storage.go)
- [Retention of state values and derived handles](../../../../../pkg/state/safestate/impl.go)
- [Async enqueue success and error paths](../../../../../pkg/pipeline/async-pipeline-impl.go)

## Construction

### Tests

- [x] Update [istructsmem/event-types_test.go](../../../../../pkg/istructsmem/event-types_test.go), [istructsmem/test_test.go](../../../../../pkg/istructsmem/test_test.go) and [istructsmem/impl_seqtrust_test.go](../../../../../pkg/istructsmem/impl_seqtrust_test.go)
  - Configure the PLog cache before constructing the test application so the no-cache case exercises storage reads
  - Exercise single and sequential log reads, check the returned CUDs, and release each event after inspecting or reapplying it

- [x] Update [storages/impl_event_storage_test.go](../../../../../pkg/sys/storages/impl_event_storage_test.go) and [storages/impl_wlog_storage_test.go](../../../../../pkg/sys/storages/impl_wlog_storage_test.go)
  - Read a real nested CUD field while the caller holds the event. Verify ownership for individual lookups, successive range callbacks, and a lookup that fails after receiving its event

- [x] Create [stateprovide/wlog_test.go](../../../../../pkg/state/stateprovide/wlog_test.go)
  - Verify that host and bundled batch reads transfer delivered events to the callback and release undelivered events on callback error. Verify cleanup after an existence check
  - Verify that a callback panic expires row and child handles, an inner read preserves outer-row handles, and a separate lookup survives until invocation cleanup

- [x] Update [command/impl_test.go](../../../../../pkg/processors/command/impl_test.go), [it/bug_test.go](../../../../../pkg/sys/it/bug_test.go) and [it/impl_test.go](../../../../../pkg/sys/it/impl_test.go)
  - Release events when collecting offsets for rejected commands. Retain batched events through the bbolt buffer-lifetime check and inspect nullable fields before releasing their WLog row

- [x] Update [basicusage/main_test.go_](../../../../../pkg/iextengine/wazero/_testdata/basicusage/main_test.go_) and [tests/wasm/main_test.go](../../../../../pkg/iextengine/wazero/_testdata/tests/wasm/main_test.go)
  - Register cleanup for each directly invoked extension's test API and give independent projector scenarios their own state

- [x] Update [actualizers/async_test.go](../../../../../pkg/processors/actualizers/async_test.go)
  - Verify release counts when a send fails partway through a batch and when cancellation follows a successful send. Each unsent event must be released once by the reader and each queued event once by the pipeline

### Log event ownership

- [x] Update [istructs/events-types.go](../../../../../pkg/istructs/events-types.go) and [istructs/interface.go](../../../../../pkg/istructs/interface.go)
  - Document that each PLog or WLog callback receives one reference, including on single-event reads, and must release it after its last use or transfer ownership

- [x] Update [istructsmem/event-dynobuf.go](../../../../../pkg/istructsmem/event-dynobuf.go)
  - Release partially decoded argument children, created records, and updated records if decoding fails before they are attached to the event. Leave attached entries to the event's cleanup
  - Release a previously decoded update before replacing it with another update for the same record ID

- [x] Update [istructsmem/impl.go](../../../../../pkg/istructsmem/impl.go)
  - In single-event PLog and WLog reads, return the borrowed byte buffer on a storage error or miss. Attach it to the event before decoding so the event can free it together with partially decoded rows on failure
  - Hand the reader reference to the callback on successful decoding or a PLog cache hit. A callback error still leaves cleanup with the callback owner
  - In `GetORec`, defer release inside the WLog callback so both a missing record and a successful record copy release the source event

- [x] Update [sys/sqlquery/impl_plog.go](../../../../../pkg/sys/sqlquery/impl_plog.go) and [sys/sqlquery/impl_wlog.go](../../../../../pkg/sys/sqlquery/impl_wlog.go)
  - Defer release on entry to each row callback so rendering, JSON encoding, and delivery of the independent result string can finish on all return paths

- [x] Update [cluster/impl_updatecorrupted.go](../../../../../pkg/cluster/impl_updatecorrupted.go)
  - Hold the source reference from either lookup until the replacement builder has copied the bytes, then release it. Also release it if the read returns an error after delivering the event
  - Pass the source bytes directly to the builder, which already copies them

- [x] Update [actualizers/async.go](../../../../../pkg/processors/actualizers/async.go)
  - Track successful sends in the shared batch loop for catch-up and single-event reads. On send failure release the current entry and the remaining entries. On cancellation after a successful send release only the remaining entries
  - Clear consumed batch slots before reusing the slice. Keep successfully queued events under workpiece ownership

### WLog state and extension handles

- [x] Update [istructs/recources-types.go](../../../../../pkg/istructs/recources-types.go), [state/interface.go](../../../../../pkg/state/interface.go) and [storages/impl_wlog_storage.go](../../../../../pkg/sys/storages/impl_wlog_storage.go)
  - Return the event from a single lookup and pass each event to the range callback. Each delivered value owns one reference and borrows its fields and CUDs from that event
  - Release the acquired event if a single lookup fails after its callback runs

- [x] Update [state/utils.go](../../../../../pkg/state/utils.go), [stateprovide/impl_host_state.go](../../../../../pkg/state/stateprovide/impl_host_state.go) and [stateprovide/impl_bundled_host_state.go](../../../../../pkg/state/stateprovide/impl_bundled_host_state.go)
  - Use `ReleaseWLogEvent` for discarded values. Transfer each batch value before calling the consumer and release undispatched events on error. Release events used only to check existence

- [x] Update [journal/impl_journal.go](../../../../../pkg/sys/journal/impl_journal.go)
  - Release each row when its journal callback returns, including skipped rows and errors during rendering or delivery

- [x] Update [isafestateapi/interface.go](../../../../../pkg/state/isafestateapi/interface.go), [safestate/impl.go](../../../../../pkg/state/safestate/impl.go) and [safestate/provide.go](../../../../../pkg/state/safestate/provide.go)
  - Track each derived WLog range handle by its owning row. When the callback exits, invalidate that row's handles and release its event
  - Have `IStateSafeAPI.Release()` release single-lookup events and invalidate value handles. Call it before reusing an API instance

- [x] Update [wazero/impl.go](../../../../../pkg/iextengine/wazero/impl.go), [exttinygo/interface.go](../../../../../pkg/exttinygo/interface.go) and [internal/impl_state_tinygo.go](../../../../../pkg/exttinygo/internal/impl_state_tinygo.go)
  - Defer release for each invocation attempt in the WebAssembly host. Document the callback lifetime for range rows and the invocation lifetime for single lookups. The TinyGo adapter leaves cleanup to the host

- [x] Update [exttinygotests/impl.go](../../../../../pkg/exttinygo/exttinygotests/impl.go)
  - Expose `Release()` on `TestAPI` for direct extension calls and register cleanup in the command and projector test constructors
