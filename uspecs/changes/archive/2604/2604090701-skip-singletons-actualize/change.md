---
registered_at: 2026-04-08T19:38:46Z
change_id: 2604081938-skip-singletons-actualize
baseline: 2d707b6e4fd9a3b4427356077ea26a1731da85ff
issue_url: https://untill.atlassian.net/browse/AIR-3532
archived_at: 2026-04-09T07:01:06Z
---

# Change request: Skip singletons in ActualizeSequencesFromPLog

## Why

`ActualizeSequencesFromPLog()` adds all new CUD IDs to the `RecordIDSequence` batch without checking for singletons. Singleton IDs are predefined (65536–66047) and below `FirstUserRecordID` (200001). Since the batcher uses `maps.Copy` to overwrite `toBeFlushed`, a singleton-only event can lower the stored sequence value, causing subsequent `Next()` calls to return IDs in the reserved range. See [issue.md](issue.md) for details.

## What

Updated `ActualizeSequencesFromPLog()` to skip singleton CUDs:

- Skip CUDs whose QName resolves to `ISingleton` via `appDef.Type()`
- Add test case with singleton CUD verifying it does not appear in the batch
- Update `sequences--arch.md` to document singleton handling during actualization
