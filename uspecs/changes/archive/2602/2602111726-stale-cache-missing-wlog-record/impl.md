# Implementation plan

## Construction

- [x] update: [istoragecache/impl_test.go](../../../../../pkg/istoragecache/impl_test.go)
  - add: Race condition test -- concurrent `Get()` and `InsertIfNotExists()` on the same key using channel-based synchronization to create deterministic interleaving, verify `Get()` does not overwrite valid data with `nil`
  - remove: test `Should remove item from cache when it was not found from underlying storage`
    - now it is not possible to test using this approach: if item exists it is always used
- [x] Review
- [x] update: [istoragecache/impl.go](../../../../../pkg/istoragecache/impl.go)
  - add: `cacheMu sync.Mutex` field to `cachedAppStorage` struct
  - update: `Get()` -- lock `cacheMu` around negative-cache write with check-then-set pattern
  - update: `getBatchFromStorage()` -- lock `cacheMu` around negative-cache writes in loop
  - update: `TTLGet()` -- lock `cacheMu` around negative-cache write with check-then-set pattern
  - update: `InsertIfNotExists()` -- lock `cacheMu` around positive-cache write
  - update: `Put()` -- lock `cacheMu` around positive-cache write
  - update: `PutBatch()` -- lock `cacheMu` around positive-cache writes in loop
  - update: `CompareAndSwap()` -- lock `cacheMu` around positive-cache write
