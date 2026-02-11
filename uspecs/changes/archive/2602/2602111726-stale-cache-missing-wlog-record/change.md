---
registered_at: 2026-02-11T15:42:52Z
change_id: 2602111642-stale-cache-missing-wlog-record
baseline: 9640efecb6d1573402ba830e58ba6698e894239f
archived_at: 2026-02-11T17:26:14Z
---

# Change request: Fix stale cached missing record in wlog query

## Why

Permanent stale "missing" cache entries for wlog records. The record exists in storage but reads by exact offset return nothing.

Symptoms:

- `select * from <wsid>.sys.wlog where offset > 16226` -- finds offset 16227
- `select * from <wsid>.sys.wlog where offset = 16227` -- returns nothing
- Other commands/queries that check wlog by ID (e.g. `GetORec`) also do not find the record
- This is a permanent error, not transient

## Root cause analysis

### Two query paths

- Equality (`offset = X`): `ReadWLog(ctx, wsid, offset, 1, cb)` -> `storage.Get()` -> cached
- Range (`offset > X`): `ReadWLog(ctx, wsid, offset, N, cb)` -> `storage.Read()` -> bypasses cache entirely

### Race condition in command processor pipeline

In `pkg/processors/command/provide.go`, the `syncProjectorsAndPutWLog` stage uses `ForkOperator(ForkSame, ...)` which runs two branches **concurrently** via goroutines (`pkg/pipeline/fork-operator-impl.go`):

- Branch 1: `DoSyncActualizer` -- executes sync projectors
- Branch 2: `PutWlog` -- writes the wlog record

### How the stale cache entry is created

Both branches share the same `cachedAppStorage` instance (single `cfg.storage` per app, set in `appstruct-types.go:147`).

`cachedAppStorage.Get()` (`impl.go:283`) is not atomic -- there is a window between `s.storage.Get()` (line 304) and `s.cache.Set()` (line 313) where another goroutine can interleave:

```text
Goroutine A (Sync Projector)              Goroutine B (PutWlog)
================================          ================================
Get(pKey, cCols, &data)
  HasGet(key) -> isCached=false
  s.storage.Get() -> ok=false
                                          InsertIfNotExists(pKey, cCols, value)
                                            s.storage.InsertIfNotExists() -> ok=true
                                            s.cache.Set(key, validData)
  s.cache.Set(key, nil)                   // overwrites valid data with "missing"
```

The cache is `VictoriaMetrics/fastcache` -- thread-safe at individual operation level, but last-writer-wins with no ordering guarantee. If the projector's `Set(key, nil)` lands after `InsertIfNotExists`'s `Set(key, validData)`, the valid entry is permanently overwritten.

Sync projectors have `Storage_WLog` with `S_GET` access (`impl_sync_actualizer_state.go:37`). `wLogStorage.Get()` calls `ReadWLog(ctx, wsid, offset, 1, cb)` which goes through `cachedAppStorage.Get()`.

### Why the error is permanent

- Subsequent equality reads use `Get()` -> finds cached `nil` -> returns `ok=false` without hitting storage
- Range reads use `Read()` -> bypasses cache -> finds the record in storage
- No cache invalidation mechanism exists for this case
- `InsertIfNotExists` only updates cache when `ok=true` (successful insert); it won't fix an already-stale entry on retry

### Affected code paths

- `pkg/istructsmem/impl.go` `GetORec()` -- reads wlog with count=1, used by projectors and other callers
- `pkg/istructsmem/impl.go` `ReadWLog()` -- branches on `toReadCount == 1` (Get) vs `!= 1` (Read)
- `pkg/istoragecache/impl.go` `cachedAppStorage.Get()` -- caches both found and "missing" entries
- `pkg/istoragecache/impl.go` `cachedAppStorage.InsertIfNotExists()` -- only updates cache on success
- `pkg/processors/command/provide.go` -- concurrent ForkOperator for sync projectors and PutWlog

## References

- [istoragecache/impl.go](../../../../../pkg/istoragecache/impl.go) -- `cachedAppStorage.Get()` (line 283), `InsertIfNotExists()` (line 126)
- [istructsmem/impl.go](../../../../../pkg/istructsmem/impl.go) -- `ReadWLog()` (line 518), `GetORec()` (line 354), `PutWlog()` (line 452)
- [istructsmem/appstruct-types.go](../../../../../pkg/istructsmem/appstruct-types.go) -- `cfg.storage` shared instance (line 147)
- [processors/command/provide.go](../../../../../pkg/processors/command/provide.go) -- concurrent `ForkOperator` for sync projectors and PutWlog (line 70)
- [pipeline/fork-operator-impl.go](../../../../../pkg/pipeline/fork-operator-impl.go) -- `ForkOperator.DoSync()` runs branches as goroutines (line 41)
- [state/stateprovide/impl_sync_actualizer_state.go](../../../../../pkg/state/stateprovide/impl_sync_actualizer_state.go) -- sync projector `Storage_WLog` with `S_GET` (line 37)
- [sys/storages/impl_wlog_storage.go](../../../../../pkg/sys/storages/impl_wlog_storage.go) -- `wLogStorage.Get()` calls `ReadWLog` with count=1 (line 99)

archived_at: 2026-02-11T17:26:14Z
---

## What

- Fix the race condition that creates stale "missing" cache entries for wlog records
