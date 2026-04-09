# AIR-3532: sequences: ActualizeSequencesFromPLog must skip singletons

- **Type**: Sub-task
- **Status**: In Progress
- **Assignee**: d.gribanov@dev.untill.com

## Why

`ActualizeSequencesFromPLog()` adds all new CUD IDs to the `RecordIDSequence` batch without checking for singletons. Singleton IDs are predefined (65536–66047) and below `FirstUserRecordID` (200001). Since the batcher uses `maps.Copy` to overwrite `toBeFlushed`, a singleton-only event can lower the stored sequence value, causing subsequent `Next()` calls to return IDs in the reserved range.

## What

- Skip singleton CUDs in `ActualizeSequencesFromPLog()` by checking `appDef.Type(cud.QName())` for `ISingleton`
- Add test case with singleton CUD verifying it does not appear in the batch
- Update `sequences--arch.md` to document singleton handling during actualization
