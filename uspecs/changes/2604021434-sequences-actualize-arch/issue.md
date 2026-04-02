# AIR-3507: Sequences: actualize architecture

**Type:** Sub-task
**Status:** In Progress
**Assignee:** d.gribanov@dev.untill.com
**URL:** https://untill.atlassian.net/browse/AIR-3507

## Description

Findings from comparing architecture doc with actual implementation:

- `IVVMSeqStorageAdapter` is not described in the design
- Typos, minor errors
- Extra fields not in design: `cleanupCtx`/`cleanupCtxCancel`, `iTime`, `transactionIsInProgress`, `retrierCfg` — all reasonable additions

### Start()

- Design says `Start()` returns next PLogOffset. The implementation returns `s.nextOffset` but does NOT increment it here — increment happens in `Flush()`. Design comment says "Start() returns 5 and increments nextOffset to 6" — but the implementation doesn't increment in Start. This is correct because each Start returns the same offset until Flush increments it
- `Start()` signals flushing on overflow (`s.signalToFlushing()`) but returns false immediately without waiting. The caller must retry Start(). The design doesn't specify this signal behavior — it just says return 0, false. The signal is an optimization

### Next()

- Design says "Read all known Numbers for wsKind, wsID" and "Write all Numbers to s.lru". Implementation reads only the single requested seqID, not all known seqIDs for the wsKind — less efficient but simpler and correct
- Does not write all numbers to cache — only writes the single requested one

### flusher()

- Retry uses `s.cleanupCtx` not `flusherCtx`. This means the flusher won't stop retrying when Actualize cancels the flusher — only when cleanup happens. This is intentional: the flusher should keep trying to write even during actualization cancellation to avoid data loss

### Actualize()

- Added: validates that Sequencing Transaction IS in progress (panics if not) — design only says "Panics if Actualization is already in progress". Medium — design interface comment says "Panics if Actualization is already in progress" but does NOT say "Panics if Sequencing Transaction is not in progress". Implementation adds this stricter requirement
- Cleans `inproc`, cache, `currentWSID`/`wsKind` in `Actualize()` instead of in `actualizer()` — design says `actualizer()` should do all cleaning. Medium — cleaning is split between `Actualize()` (`inproc`, cache, `currentWSID`/`wsKind`) and `actualizer()` (`toBeFlushed`, `toBeFlushedOffset`, `nextOffset`). Design puts ALL cleaning in `actualizer()`

