---
registered_at: 2026-01-30T11:20:01Z
change_id: 260130-differ-cached-missing-key-vs-empty-value
baseline: 3410f3e203d4f46303e943753dd9f21f9f2a0b35
archived_at: 2026-01-30T15:26:00Z
---

# Change request: Differ cached missing key vs empty value in istoragecache

## Why

The current implementation in `pkg/istoragecache/impl.go` cannot distinguish between a cached "key not found" state and a cached "key exists with empty value" state. When `Get` or `GetBatch` fetches from storage and the key is not found, it caches an empty `DataWithExpiration`. On subsequent cache hits, it returns `ok=false` despite the value actually exists but is empty.

## What

Implement a mechanism to differentiate between:

- Cached information that the key is missing in storage (should return `ok=false`)
- Cached information that the key exists in storage but has an empty value (should return `ok=true` with empty data)

Affected functions in `pkg/istoragecache/impl.go`:

- `Get` - currently returns `ok=false` for cached empty values, should return `ok=true`
- `GetBatch` / `getBatchFromCache` - same issue with `item.Ok` handling
- `TTLGet` - has partial handling using `len(*data) != 0` check but still has the same fundamental issue
