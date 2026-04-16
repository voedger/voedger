# AIR-3532: sequences: Singleton IDs corrupt RecordID sequence after recovery

- **Type:** Sub-task
- **Status:** In Progress
- **Assignee:** d.gribanov@dev.untill.com

## Why

During PLog actualization (on startup or after error), `ActualizeSequencesFromPLog` collects record IDs from ALL new CUDs, including singletons. Singleton CDocs receive their IDs from the singletons table (65536+), bypassing both `IDGenerator.NextID()` and `sequencer.Next()` during normal command processing. However, during actualization these singleton IDs are stored into the sequencer state (`toBeFlushed`) as the last known value for the RecordID sequence.

When a regular CDoc is inserted after recovery, `sequencer.Next()` finds the singleton ID (e.g., 65537) in its state and returns 65538. Meanwhile, `IDGenerator.NextID()` correctly returns 200001 (`FirstUserRecordID`). The mismatch causes a panic in `implIDGeneratorReporter.NextID`.

The root cause is that `sequencer.Next()` does not enforce the `initialValue` floor when determining the next number. The `initialValue` is only consulted when the stored number is exactly 0 (never written). Values obtained from cache, inproc, or `toBeFlushed` are used as-is, even if they are below `initialValue`.

## What

- Implement a test that shows the problem
- In `sequencer.incrementNumber()`, add `initialValue` parameter and apply `max(number+1, initialValue)` instead of `number+1`
- Update all 4 call sites in `sequencer.Next()` to pass `initialValue`
- Remove the `if nextNumber == 0 { nextNumber = initialValue - 1 }` special case in the storage path of `Next()` — it becomes redundant since `incrementNumber` handles it uniformly
