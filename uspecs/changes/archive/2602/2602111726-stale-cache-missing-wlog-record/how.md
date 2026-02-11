# How: Fix stale cached missing record in wlog query

## Approach

- Add `sync.Mutex` field (`cacheMu`) to `cachedAppStorage` in `istoragecache/impl.go`
- In `Get()`: lock `cacheMu` around the negative-cache write, check `HasGet` first, only write `nil` if key is still absent
- In `GetBatch()` / `getBatchFromStorage()`: same pattern for negative-cache writes in the loop
- In `TTLGet()`: same pattern for the `cache.Set(key, nil)` call
- In all write methods (`InsertIfNotExists`, `Put`, `PutBatch`, `CompareAndSwap`): lock `cacheMu` around `cache.Set` calls
- This makes the check-then-set in `Get()` atomic with respect to positive writes from other methods
- Lock is held only for in-memory `fastcache` operations (nanoseconds), no storage I/O under the lock
- Add a test in `istoragecache/` that reproduces the race: concurrent `Get()` and `InsertIfNotExists()` on the same key using `SetTestDelayGet` from `istorage/mem/impl.go` to widen the race window

Invariant: `Get()` must never write `nil` to cache after a write method has written valid data for the same key

Negative-cache write pattern in `Get()`:

```go
s.cacheMu.Lock()
_, alreadyCached := s.cache.HasGet(nil, key)
if !alreadyCached {
    s.cache.Set(key, nil)
}
s.cacheMu.Unlock()
```

Positive-cache write pattern in write methods:

```go
s.cacheMu.Lock()
s.cache.Set(key, validData)
s.cacheMu.Unlock()
```

Methods that need `cacheMu` around `cache.Set`:

- `Get` -- negative write (check-then-set)
- `getBatchFromStorage` -- negative writes in loop
- `TTLGet` -- negative write
- `InsertIfNotExists` -- positive write
- `Put` -- positive write
- `PutBatch` -- positive writes in loop
- `CompareAndSwap` -- positive write

Methods that do NOT need changes:

- `Read` -- bypasses cache entirely
- `TTLRead` -- bypasses cache entirely
- `getBatchFromCache` -- read-only, no `Set` calls

## Performance influence

Critical section contains only in-memory `fastcache` operations (`HasGet` + `Set`), no storage I/O. Estimated hold time per lock acquisition: 50-100ns.

At 4K command ops/sec (one op every 250us), lock overhead per op is < 0.1% of the time budget. Multiple `Get`/`Put` calls per op increase the number of acquisitions but each is nanoseconds -- total lock time remains negligible.

Contention analysis:

- Uncontended `sync.Mutex` lock/unlock: ~20-30ns on modern hardware
- Contended case requires two goroutines to call `cache.Set` for the same `cachedAppStorage` at the same nanosecond-scale window -- probability is low given the short critical section
- `PutBatch` holds the lock for N iterations in a loop -- for typical batch sizes (tens of items) this is still sub-microsecond
- Cache reads (`HasGet` without `Set`) in the fast path of `Get()` and `getBatchFromCache()` do not acquire the lock at all

No measurable throughput or latency impact expected at current scale.

Alternatives considered:

- Remove negative caching entirely -- simplest (one-line fix) but loses the optimization for repeated lookups of genuinely absent keys
- Lock only in `Get()` without locking write methods -- reduces race window to nanoseconds but not provably correct
- Per-key striped lock -- correct but adds unnecessary complexity for 4K ops/sec throughput

References:

- [istoragecache/impl.go](../../../../../pkg/istoragecache/impl.go)
- [istorage/mem/impl.go](../../../../../pkg/istorage/mem/impl.go)
