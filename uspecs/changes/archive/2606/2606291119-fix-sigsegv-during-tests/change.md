---
change_id: 2606290807-fix-sigsegv-during-tests
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-4355
domains: [prod]
scope: [storage]
---

# Change request: Fix SIGSEGV during async actualizer CUD logging

Refs:

- [AIR-4355: voedger: fix SIGSEGV during tests](./issue-AIR-4355.md)

## Why

The airsbp3 test suite aborts intermittently with a fatal SIGSEGV, breaking CI and masking other failures. The crash stems from sequentially read log events referencing transient, memory-mapped storage memory that is invalidated once the read transaction closes, so any deferred access (such as verbose CUD logging in the async actualizer) reads freed memory.

## What

Symptom: The async actualizer's verbose CUD logging faults with a fatal SIGSEGV while reading a record's fields, aborting the test process.

```text
async actualizer batch-reads PLog (ReadToTheEnd)
      |
      v
ReadPLog/ReadWLog sequential branch (istructsmem/impl.go)   <-- fault: builds event via loadFromBytes over transient storage-callback bytes; event does not own them
      |
      v
storage read transaction closes; bbolt remaps the mmap region
      |
      v
asyncProjector.DoAsync logs CUDs
  -> FieldsToMap -> rowType.SpecifiedValues -> dynobuffers.IterateFields
      |
      v
flatbuffers reads dangling mmap memory -> SIGSEGV   (symptom)
```

Corrected behavior: Sequentially read PLog/WLog events own a private, pooled copy of their bytes, so they stay valid after the storage read callback returns and deferred CUD logging no longer faults.

## How

Decisions:

- Fix at the `istructsmem` layer, not per driver: the sequential `ReadPLog`/`ReadWLog` branch builds events via `newStoredEvent`, which copies the storage-callback bytes into a pooled buffer the event owns (`pkg/istructsmem/event-types.go`), so events stay valid after the read transaction closes regardless of any driver's buffer-lifetime contract
- Only `bbolt` can trigger the defect: its `Read` hands the callback a zero-copy view into the memory-mapped file, valid only inside the read transaction; `mem`, `cas` and `amazondb` copy into Go-managed memory, so they are safe
- Reproduce with a real-`bbolt` integration test (no driver/engine instrumentation): write events, sequentially read them with `ReadToTheEnd` while retaining the returned events, then drive additional storage writes that free the event pages and make `bbolt` grow and re-map its file; afterwards the retained events read freed/remapped memory on the unfixed code and stay valid on the fixed code
- Place the reproduction in `sys_it` with an owned VIT config that provides `bbolt` as the storage, so `istructsmem` keeps no dependency on a storage driver; drive writes through the public command pipeline (`c.sys.CUD`) and read via `Events().ReadWLog(...ReadToTheEnd)` so the test exercises the actual fixed code path

Out of scope:

- Hardening `bbolt`'s `TTLGet`, which (unlike every other driver) returns a zero-copy mmap view; no current path retains it, so it is tracked separately
- Bumping the `voedger` dependency in `airs-bp3` to pick up the fix

References:

- [sequential PLog/WLog read fix](../../../../../pkg/istructsmem/impl.go)
- [event that owns its bytes](../../../../../pkg/istructsmem/event-types.go)
- [bbolt read hands back mmap views](../../../../../pkg/istorage/bbolt/impl.go)
- [transient ReadCallback contract](../../../../../pkg/istorage/interface.go)
- [sys_it reproduction test](../../../../../pkg/sys/it/bug_test.go)

## Construction

- [x] update: [sys/it/bug_test.go](../../../../../pkg/sys/it/bug_test.go)
  - add: `TestBug_BatchedLogEventsMustOwnTheirBytes` - a real-`bbolt` VIT integration test that posts `c.sys.CUD` events, sequentially reads the whole WLog (`ReadToTheEnd`) while retaining the returned events, then churns the same storage via the public API so `bbolt` frees pages and grows/re-maps its file, and finally asserts the retained events still expose their original CUD field values
  - add: `getBboltVITCfg` helper that builds a `NewOwnVITConfig` wiring `bbolt.Provide` over a temp dir as the `StorageFactory`

- [x] update: [istructsmem/event-types_test.go](../../../../../pkg/istructsmem/event-types_test.go)
  - add: white-box assertion in the sequential-read subtest that the read event owns its `buffer`

- [x] update: [istructsmem/event-types.go](../../../../../pkg/istructsmem/event-types.go)
  - add: `newStoredEvent` helper that builds an event owning a private, pooled copy of the storage-callback bytes, so the event stays valid after the read transaction closes

- [x] update: [istructsmem/impl.go](../../../../../pkg/istructsmem/impl.go)
  - update: the sequential (`ReadToTheEnd`) branch of `ReadPLog` and `ReadWLog` to build events via `newStoredEvent` instead of `loadFromBytes` over the transient storage-callback bytes
