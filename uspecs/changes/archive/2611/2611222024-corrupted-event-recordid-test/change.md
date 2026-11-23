---
change_id: 2611221953-corrupted-event-recordid-test
type: test
issue_url: https://untill.atlassian.net/browse/AIR-5314
domains: [prod]
scope: [apps]
---

# Change request: RecordID recovery after WLog event corruption

Refs:

- [AIR-5314: RecordID recovery coverage after event corruption](./issue-AIR-5314.md)

## Why

Marking the last ID-bearing WLog event as corrupted can hide the last issued RecordID from workspace recovery. When no later ID-bearing event exists, recovery can reuse an already allocated ID, causing a subsequent insert to fail under the trust level that checks new record writes. Capture this behavior in a specification and integration coverage.

## What

- A Gherkin scenario describes insertion after workspace recovery when the last ID-bearing WLog event is corrupted and no later ID-bearing event exists, with sequence trust level 0 (do not trust new record writes)
- A corresponding integration test verifies that inserting an ordinary record returns HTTP 500 with a message containing `sequences violation`, without automatic retries hiding the failure
- Existing runtime behavior is preserved; this change adds coverage of the failure without introducing a corruption guard or override header

## How

Decisions:

- Use an isolated VIT integration fixture with operator-facing VSQL corruption and a storage-preserving server restart to exercise the actual persistence, recovery, and HTTP response paths
- Preserve the record created by the corrupted event and use a later non-ID-bearing partition tail, preventing last-event replay from repairing the damaged WLog entry while retaining the record that causes the collision
- Establish recovery readiness through a command that allocates no RecordIDs, allowing transient recovery responses to retry during setup and disabling retries for the insert under test

Assumptions:

- None

References:

- [Workspace recovery integration patterns and non-ID-bearing commands](../../../../../pkg/sys/it/impl_recovery_test.go)
- [Operator-facing log corruption coverage](../../../../../pkg/sys/it/impl_vsqlupdate_test.go)
- [Storage-preserving server restart](../../../../../pkg/vit/restart.go)
- [Partition replay and workspace sequence recovery](../../../../../pkg/processors/command/impl.go)
- [Integration harness retry policy](../../../../../pkg/vit/consts.go)

## Functional design

- [x] update: [apps/recovery.feature](../../../../specs/prod/apps/recovery.feature)
  - update: Feature description to distinguish recovery from invalid checkpoints from loss of record-ID allocation information through event corruption
  - add: Scenario "An insert fails after recovery when the last ID-bearing WLog event is corrupted" under "Workspace recovery", using an earlier valid ID-bearing event, a later corrupted ID-bearing event whose ordinary record remains stored, and a non-ID-bearing log tail with no later ordinary ID allocations
  - specify: Sequence trust level 0 and completed VVM restart, partition recovery, and workspace recovery as preconditions; inserting one ordinary record without automatic retries returns "500 Internal Server Error" with a response message containing "sequences violation"

## Construction

- [x] update: [sys/it/impl_recovery_test.go](../../../../../pkg/sys/it/impl_recovery_test.go)
  - add: `TestWorkspaceRecoveryAfterWLogCorruption` with subtest "recovery: scn: An insert fails after recovery when the last ID-bearing WLog event is corrupted" and verbatim step comments matching [apps/recovery.feature](../../../../specs/prod/apps/recovery.feature)
  - reuse: An owned VIT fixture, existing workspace and command helpers, subtest-local assertions, and teardown of the current VVM; verify sequence trust level 0 and the specified history with ordinary IDs 200001 and 200002 at WLog offsets 4 and 5 after three initialization events, followed by a non-ID-bearing event at offset 6
  - verify: The partition's last PLog event corresponds to WLog offset 6, and asynchronous projectors have flushed before restart; mark WLog event 5 as corrupted through VSQL and confirm that record 200002 remains stored
  - recover: Restart the VVM over the same storage and complete recovery using the existing non-ID-bearing command with retries for temporary recovery responses; verify that the corrupted event and original record survive, and that no later event allocates ordinary IDs
  - assert: Submit one ordinary-record insert with `httpu.WithNoRetryPolicy()` and check HTTP 500 plus response error text containing `sequences violation`, exercising the real record collision without injected errors
  - validate: Run `gofmt` on the changed test file and `go test ./pkg/sys/it -run '^(TestWorkspaceRecoveryAfterWLogCorruption|TestWorkspaceSequencesAfterRecovery|TestVSqlUpdate_BasicUsage_Corrupted)$' -count=1`, then review the diff for scenario coverage, assertion scope, and VVM cleanup
