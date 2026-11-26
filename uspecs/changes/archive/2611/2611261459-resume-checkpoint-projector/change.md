---
change_id: 2611261306-resume-checkpoint-projector
type: perf
issue_url: https://untill.atlassian.net/browse/AIR-5369
domains: [prod]
scope: [apps]
---

# Change request: Recovery checkpoint projector startup after initial recovery

Refs:

- [AIR-5369: voedger: resume sys.ApplyRecoveryCheckpoint from the PLog offset reached by initial recovery](./issue-AIR-5369.md)

## Why

Initial recovery scans the entire PLog and saves its last offset as the partition checkpoint. A first run of `sys.ApplyRecoveryCheckpoint` can replay the same events and overwrite that checkpoint with older offsets. For example, after recovery saves offset 250, the projector writes 100 and then 200.

## What

- For a nonempty partition, once initial recovery has persisted its workspace checkpoints and before it marks the partition complete, it sets the saved position of `sys.ApplyRecoveryCheckpoint` to the last PLog offset scanned only if the position is missing, `0`, or `FirstOffset`
- Each successful projector position write emits one `INFO` log at stage `bootstrap.initial_recovery` with message `ap.sys.ApplyRecoveryCheckpoint offset set to %d`, using the offset just written
- With that position saved, the actualizer starts at the next PLog event. A saved position greater than `FirstOffset` is preserved so the actualizer resumes from its own progress
- Command behavior after recovery stays the same, including the next PLog offset, WLog offset, and ordinary record ID issued

## How

Decisions:

- Store the projector position in the application's existing projection offsets view. Share the partition-and-projector key and offset lookup with the actualizer so both read the same row in the null workspace
- Use the registered application's view records for the position write. Verify the offset through `ActualizerOffset` on that application immediately after `InitializeCheckpoints` returns, before partition deployment
- Treat a projector position read or write error as a partition initialization failure. The missing completion marker makes the next bootstrap retry the scan, even if the projector position was saved before the marker write failed
- Verify the later insert separately through its issued numbers and the resulting workspace checkpoint. Retain overwrite coverage for a projector catching up from an established position behind the recovery checkpoint

Assumptions:

- None

Out of scope:

- Backfilling projector positions for partitions that already have an initial recovery completion marker

References:

- [Actualizer position storage and PLog resume behavior](../../../../../pkg/processors/actualizers/async.go)
- [Partition completion marker and retry rules](../../../../../pkg/checkpoints/initial_recovery.go)
- [Partition deployment after initial recovery](../../../../../pkg/btstrp/impl.go)
- [Recovery checkpoint overwrite and command recovery scenarios](../../../../../uspecs/specs/prod/apps/recovery.feature)

## Functional design

- [x] Update [partition and workspace recovery scenarios](../../../../specs/prod/apps/recovery.feature)
  - For a recovered PLog tail of 250 and an initial projector position of missing, `0`, or `FirstOffset`, verify saved position 250 before the actualizer starts. The position lookup returns 250 and the partition has exactly one position-set log. The later insert receives WLog offset 15 and record ID 200006 and advances the workspace checkpoint
  - For an established projector position of 2, verify it is still 2 when initial recovery completes at 250. Catch-up writes partition checkpoints at 100 and 200, and the next insert after restart receives PLog offset 251, WLog offset 15, and record ID 200006
  - Cover the empty PLog with a zero partition checkpoint and an absent projector position. In normal initial partition recovery, verify the partition with an existing completion marker retains its saved `FirstOffset` position before deployment
  - Extend the existing initial recovery failure and retry scenarios to projector position reads and writes. Bootstrap logs the failed operation for the affected partition, leaves it undeployed without a completion marker, and deploys it when a later attempt succeeds

## Technical design

- [x] Update [partition and workspace recovery design](../../../../specs/prod/apps/recovery--td.md)
  - Describe how initial recovery and the actualizer address the same row in the application's `sys.projectionOffsets` view using the null workspace, partition ID, and projector QName
  - Specify the write order after a nonempty PLog scan. Persist the workspace checkpoints, set the position of `sys.ApplyRecoveryCheckpoint` to the scanned tail if its saved position is absent, `0`, or `FirstOffset`, then write the partition completion marker. Preserve greater positions and skip the position lookup for an empty PLog or an existing marker
  - Specify the saved position at the point where initial recovery returns the partition as deployable. For a newly initialized position, both the backing row and the lookup through the registered application's cache contain the recovered tail. Trace the later insert and established-projector catch-up separately
  - Show how a position read or write failure leaves the partition undeployed and eligible for retry. If the position was saved but the marker write failed, the next bootstrap rescans and applies the same rule to the persisted position. Reference the logging design for the operation names
- [x] Update [application deployment architecture](../../../../specs/prod/apps/arch-deployment.md)
  - Update the checkpoint initializer's responsibilities to include persisting the projector position before partition deployment starts the actualizer. Link to the recovery design for the offset rules
- [x] Update [bootstrap logging design](../../../../specs/prod/apps/logging--td.md)
  - Add `read projector position` and `write projector position` to the operations reported at `bootstrap.initial_recovery.error`, with the application, partition ID, and underlying cause

## Construction

### Tests

- [x] Update [checkpoints/initial_recovery_test.go](../../../../../pkg/checkpoints/initial_recovery_test.go)
  - Warm the actualizer's position lookup with the offset missing, zero, or at `FirstOffset`. Recover a PLog ending at 250 and check that `ActualizerOffset` returns 250 on the same application immediately after `InitializeCheckpoints` succeeds
  - Use the same offset assertion for a one-event PLog, an empty PLog, an established position, and a partition with an existing completion marker

- [x] Update [sys/it/impl_recovery_test.go](../../../../../pkg/sys/it/impl_recovery_test.go)
  - After deployment with the projector at 250, submit the first insert into Orders with retries for 503. Verify 200 OK, WLog offset 15, record ID 200006, PLog event 251, and the eventual workspace checkpoint of 15
  - Extend the existing bootstrap failure and retry scenarios to position reads and writes. Verify the error log, the affected partition's missing marker and exclusion, deployment of healthy partitions, and recovery of the excluded partition on the next bootstrap

- [x] Update [checkpoints/impl_test.go](../../../../../pkg/checkpoints/impl_test.go)
  - Align the existing catch-up case with the established position of 2. Verify the partition checkpoint writes at 100 and 200. Reuse the existing command recovery test for the subsequent insert

### Implementation

- [x] Create [appdef/sys/projection_offsets.go](../../../../../pkg/appdef/sys/projection_offsets.go)
  - Share the `ProjectionOffsetsView` row key for a partition and projector in `istructs.NullWSID`
  - Provide a position lookup that maps a missing row to `istructs.NullOffset` and propagates other errors, and a write through the supplied application view records using the same key

- [x] Update [actualizers/async.go](../../../../../pkg/processors/actualizers/async.go)
  - Delegate `ActualizerOffset` to the shared lookup and keep ordinary position saves in the same flush as projector intents

- [x] Update [checkpoints/initial_recovery.go](../../../../../pkg/checkpoints/initial_recovery.go)
  - After a nonempty recovery has saved its workspace checkpoints, set `QNameApplyRecoveryCheckpoint` to the recovered PLog tail through the registered application's view records if its saved position is missing, `0`, or `FirstOffset`. Preserve greater positions and skip position access for an empty PLog or an existing marker
  - Immediately after a successful write, emit the specified position-set log with the written offset and partition context. Return position failures as `read projector position` or `write projector position` to the existing bootstrap handler. Persist the partition completion marker last
