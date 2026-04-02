# Implementation plan: Sequences: actualize architecture

## Technical design

- [x] update: [apps/sequences--arch.md](../../../../specs/prod/apps/sequences--arch.md)
  - already done: `IVVMSeqStorageAdapter` added to Key components in previous PR
  - add: extra `sequencer` fields not in design: `cleanupCtx`/`cleanupCtxCancel`, `iTime`, `transactionIsInProgress`, `retrierCfg`
  - fix: `Start()` description — does not increment `nextOffset`; increment happens in `Flush()`
  - add: `Start()` overflow signal behavior — signals flusher and returns false; caller must retry
  - add: `Start()` additional panics — cleanup state, unknown wsKind
  - fix: `Next()` description — reads only the single requested `seqID`, not all known seqIDs for workspace
  - fix: `Actualize()` panic condition — also panics if sequencing transaction is NOT in progress
  - fix: `Actualize()` intentionally does NOT clean `toBeFlushed` (race avoidance with flusher)
  - fix: state cleaning split — `Actualize()` cleans `inproc`/cache/`currentWSID`/`wsKind`; `actualizer()` cleans `toBeFlushed`/`toBeFlushedOffset`/`nextOffset`
  - add: `Flush()` increments `nextOffset` — not shown in sequencing transaction diagram
  - add: `New()` bootstrap trick — sets `transactionIsInProgress = true` to allow initial `Actualize()`
  - fix: goroutine lifecycle diagram — show that `Actualize()` clears cache/inproc/finishes transaction before starting actualizer goroutine
  - fix: `flusher()` retries use `s.cleanupCtx` not `flusherCtx` — intentional to avoid data loss during actualization
  - fix: interface.go `Actualize()` flow comment — flusher cancellation is in `actualizer()`, not `Actualize()`; actualizer reads PLogOffset, doesn't write it
- [x] fix: code comment bugs
  - fix: types.go lines 63-66 — comment says "Start() returns 5 and increments nextOffset to 6" but Start() does NOT increment
  - fix: impl.go Next() comment lines 187-189 — says "Read all known Numbers" / "Write all Numbers to s.cache" but reads single seqID only
  - fix: interface.go Actualize() comment lines 73-78 — flow says "Cancel and wait Flushing" but that happens in actualizer(), not Actualize()
