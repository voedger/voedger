---
change_id: 2611230205-corrupted-plog-recovery-test
type: fix
domains: [prod]
scope: [apps]
---

# Change request: Stored-event deserialization and corrupted PLog recovery coverage

## Why

Partition recovery reapplies the final PLog event. Deserializing an invalid event currently returns before marking it as stored, so recovery incorrectly rejects a corrupted event read from PLog as an unstored event. Correct the storage flag and capture the subsequent invalid-event panic from record application separately from record-ID loss after WLog corruption.

## What

- A Gherkin scenario describes the invalid-event panic when records from a corrupted final PLog event are replayed after restart
- An in-process VIT test reads the corrupted final PLog event after restart and verifies the invalid-event panic from synchronous record replay
- Successfully deserialized events are marked as stored regardless of event validity, while decoding failures remain unmarked
- Regression tests exercise storage reads without the PLog cache and reject premature marking on decoding errors

## How

Decisions:

- Mark the event as stored after successful decoding in `loadFromBytes`, covering the early returns in `loadEvent`
- Use VSQL corruption and a storage-preserving VVM restart to exercise persisted-event deserialization
- Read the final PLog event with a scan, obtain its reapplier, and assert the `ApplyRecords` panic synchronously in the test goroutine
- Map the replay scenario to a named `recovery: scn:` subtest and verbatim Gherkin step comments
- Limit coverage to record replay after restart; asynchronous partition recovery and server termination are outside this test's scope

Assumptions:

- None

References:

- [Storage-preserving restart tests](../../../pkg/vit/restart_test.go)
- [Partition replay](../../../pkg/processors/command/impl.go)
- [Record application validity check](../../../pkg/istructsmem/impl.go)

## Functional design

- [x] update: [apps/recovery.feature](../../specs/prod/apps/recovery.feature)
  - Add the scenario "Replaying records from a corrupted final PLog event after restart panics"
  - Specify the corrupted partition tail, completed restart, record-replay action, and exact observable panic

## Construction

- [x] create: [istructsmem/event-storage_test.go](../../../pkg/istructsmem/event-storage_test.go)
  - Add regression coverage for the stored flag and reapplier admission after reading valid, build-error, and corrupted events from PLog
  - Exercise single-event reads and scans using bytes placed directly in test storage so the cache cannot mask deserialization
  - Verify that decoding failures leave the event unmarked

- [x] update: [sys/it/impl_recovery_test.go](../../../pkg/sys/it/impl_recovery_test.go)
  - Align `TestCorruptedPLogRecordReplayAfterRestart` with the Gherkin scenario through its subtest name and verbatim step comments
  - Keep the owned VIT, real VSQL corruption, restart over retained storage, and synchronous panic assertion
  - Verify that the corrupted event remains the partition tail, is accepted by `GetEventReapplier`, and causes the expected invalid-event panic when `ApplyRecords` runs synchronously
  - Validate scenario structure and step mappings, run the aligned test, format the changed Go file, and lint the affected package

- [x] update: [istructsmem/event-types.go](../../../pkg/istructsmem/event-types.go)
  - Mark events as stored only after successful deserialization

- [x] update: [istructsmem/event-dynobuf.go](../../../pkg/istructsmem/event-dynobuf.go)
  - Remove the stored-flag assignment that only covers valid events
