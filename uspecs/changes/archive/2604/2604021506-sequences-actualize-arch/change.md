---
registered_at: 2026-04-02T14:34:04Z
change_id: 2604021434-sequences-actualize-arch
baseline: e46d994a8401c8264180ceda69e8992a276c5ccb
issue_url: https://untill.atlassian.net/browse/AIR-3507
archived_at: 2026-04-02T15:06:19Z
---

# Change request: Sequences: actualize architecture

## Why

The `ISequencer` architecture documentation contains discrepancies compared to the actual implementation: missing components, incorrect descriptions of method behaviors, and fields not documented. The architecture spec needs to be actualized to accurately reflect what was built.

See [issue.md](issue.md) for details.

## What

Actualize `specs/prod/apps/sequences--arch.md` to match the actual `pkg/isequencer` implementation:

- Document `IVVMSeqStorageAdapter` (currently missing from design)
- Document extra fields not in design: `cleanupCtx`/`cleanupCtxCancel`, `iTime`, `transactionIsInProgress`, `retrierCfg`
- Fix `Start()` description: does not increment `nextOffset` — increment happens in `Flush()`
- Document `Start()` overflow signal behavior: signals flusher and returns false for caller to retry
- Fix `Next()` description: reads only single requested `seqID`, not all known seqIDs for workspace
- Fix `Actualize()` description: panics if sequencing transaction is not in progress (stricter than documented)
- Fix cleaning split: `Actualize()` cleans `inproc`/cache/currentWSID/wsKind; `actualizer()` cleans `toBeFlushed`/`toBeFlushedOffset`/`nextOffset`
