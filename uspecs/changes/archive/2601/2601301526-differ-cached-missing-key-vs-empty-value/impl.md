# Implementation plan: Differ cached missing key vs empty value in istoragecache

## Technical design

Cache-only approach using `nil` vs `DataWithExpiration`:

- Store `nil` in cache to represent "key is missing in storage"
- Store `DataWithExpiration.ToBytes()` (even with empty data) to represent "key exists"
- On cache hit: if cached value length is 0 → return `ok=false`; otherwise parse as `DataWithExpiration` → return `ok=true`
- No changes to storage drivers or `DataWithExpiration` type

## Construction

- [x] update: `[istoragecache/impl.go](../../pkg/istoragecache/impl.go)`
  - Fix `Get()`: on cache hit with empty value → return `ok=false`; on cache hit with data → parse and return `ok=true`
  - Fix `Get()`: on cache miss, if storage returns `ok=false` → cache `nil`, else cache `DataWithExpiration`
  - Fix `getBatchFromCache()`: check for empty cached value to determine `item.Ok`
  - Fix `getBatchFromStorage()`: cache `nil` for missing keys, `DataWithExpiration` for existing keys
  - Fix `TTLGet()`: on cache hit with empty value → return `ok=false`; on cache hit with data → parse and return `ok=true`
  - Fix `TTLGet()`: on cache miss, cache the result from storage appropriately

- [x] update: `[istoragecache/impl_test.go](../../pkg/istoragecache/impl_test.go)`
  - Add test case: Get returns ok=false for cached missing key
  - Add test case: Get returns ok=true with empty data for cached empty value
  - Add test case: GetBatch handles both scenarios correctly
  - Add test case: TTLGet handles both scenarios correctly

- [x] Review
