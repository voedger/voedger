---
registered_at: 2026-04-14T13:26:50Z
change_id: 2604141326-seq-singleton-id-recovery
baseline: acfb394b2f2c941f0bd173b354b4e83cda8b894d
issue_url: https://untill.atlassian.net/browse/AIR-3532
archived_at: 2026-04-14T13:49:01Z
---

# Change request: Singleton IDs corrupt RecordID sequence after recovery

## Why

During PLog actualization, `ActualizeSequencesFromPLog` collects record IDs from all CUDs including singletons. Singleton CDocs receive IDs from the singletons table (65536+), bypassing the sequencer during normal processing. After recovery, `sequencer.Next()` returns a value based on the singleton ID instead of the correct RecordID sequence value, causing a panic due to mismatch with `IDGenerator.NextID()`. The root cause is that `sequencer.Next()` does not enforce the `initialValue` floor when determining the next number.

See [issue.md](issue.md) for details.

## What

- Implement a test that demonstrates the problem
- Fix `sequencer.incrementNumber()` to enforce the `initialValue` floor
  - Add `initialValue` parameter to `incrementNumber()` and apply `max(number+1, initialValue)` instead of `number+1`
  - Update all 4 call sites in `sequencer.Next()` to pass `initialValue`
  - Remove the `if nextNumber == 0 { nextNumber = initialValue - 1 }` special case in the storage path of `Next()` — it becomes redundant since `incrementNumber` handles it uniformly
- make according updates to sequences--arch.md
