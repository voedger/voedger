---
change_id: 2611230917-recovery-checkpoint-overwrite
type: test
issue_url: https://untill.atlassian.net/browse/AIR-5299
domains: [prod]
scope: [apps]
---

# Change request: Recovery checkpoint overwrite coverage

Refs:

- [AIR-5299: document that ApplyRecoveryCheckpoint projector will overwrite the initial checkpoint on first run](./issue-AIR-5299.md)

## Why

Initial recovery can store a partition checkpoint at the PLog tail before a newly introduced `sys.ApplyRecoveryCheckpoint` projector begins processing from `FirstOffset`. The projector normally replaces that checkpoint at each 100-event boundary, and the bounded tail left behind is safely read again during later recovery, but this behavior is not explicitly specified or covered by tests.

## What

Recovery checkpoint behavior gains specification and regression coverage:

- A Gherkin scenario describes the first projector run replacing an initial partition checkpoint while processing the PLog from `FirstOffset`
- Corresponding tests verify that `sys.ApplyRecoveryCheckpoint` stores each 100-event boundary even when this temporarily moves the partition checkpoint backward
- Recovery coverage verifies that the final checkpoint can remain within the last 100 PLog events and that rereading this tail preserves checkpoint bookkeeping

## How

Decisions:

- Use layered coverage: focused projector tests prove checkpoint writes at 100-event boundaries, while command processor coverage proves bounded-tail recovery and subsequent number allocation
- Use command processor recovery coverage for the bounded-tail behavior, reusing its in-memory checkpoint storage and deterministic PLog read observation
- Verify bounded-tail safety through observable recovery results after restart, including correct PLog offset, WLog offset, and record ID allocation, rather than treating checkpoint values alone as proof of correctness

Assumptions:

- None

References:

- [recovery behavior specification](../../../../specs/prod/apps/recovery.feature)
- [recovery checkpoint design](../../../../specs/prod/apps/recovery--td.md)
- [checkpoint projector implementation](../../../../../pkg/checkpoints/impl.go)
- [checkpoint projector component coverage](../../../../../pkg/checkpoints/impl_test.go)
- [command processor recovery coverage](../../../../../pkg/processors/command/recovery_checkpoint_test.go)

## Functional design

- [x] update: [apps/recovery.feature](../../../../specs/prod/apps/recovery.feature)
  - add: scenario for the first recovery checkpoint projector run replacing the initial partition checkpoint at 100-event boundaries, leaving a bounded PLog tail, and recovery rereading that tail without changing the numbers issued by the next insert

## Construction

- [x] update: [checkpoints/impl_test.go](../../../../../pkg/checkpoints/impl_test.go)
  - extend projector component coverage with an initial partition checkpoint at offset 250 and representative events through offset 250
  - verify successful partition-checkpoint writes at offsets 100 and 200 and the final stored checkpoint at offset 200

- [x] update: [processors/command/recovery_checkpoint_test.go](../../../../../pkg/processors/command/recovery_checkpoint_test.go)
  - add command processor coverage for recovery after the projector has overwritten the initial checkpoint
  - create PLog history through offset 250 and workspace history through WLog offset 14 with last allocated ordinary record ID 200005
  - store the projector result at offset 200, restart the command processor, and verify recovery reads once from offset 200
  - verify the next insert writes PLog offset 251, WLog offset 15, and record ID 200006
  - validate the Gherkin mapping and run the focused checkpoint and recovery tests
