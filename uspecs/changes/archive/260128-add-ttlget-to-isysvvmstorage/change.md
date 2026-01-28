---
registered_at: 2026-01-28T07:40:47Z
change_id: 260128-add-ttlget-to-isysvvmstorage
baseline: 131ee4b938a06cad88c47c9b9bec590457bb17cf
archived_at: 2026-01-28T07:51:42Z
---

# Add TTLGet to ISysVvmStorage

## Problem

`IAppTTLStorage.TTLGet()` currently calls `ISysVvmStorage.Get()` which does not properly check TTL expiration through the cache layer.

The cache layer (`cachedAppStorage` in `pkg/istoragecache/impl.go`) has a `TTLGet()` method that:

1. Checks if cached data is expired using `d.IsExpired(s.iTime.Now())`
2. Deletes expired entries from cache
3. Falls back to underlying storage's `TTLGet()`

However, `ISysVvmStorage` interface only exposes `Get()`, not `TTLGet()`. This means when `implAppTTLStorage.TTLGet()` calls `s.sysVVMStorage.Get()`, it bypasses the cache layer's TTL-aware logic.

## Solution

Add `TTLGet()` method to `ISysVvmStorage` interface and update `implAppTTLStorage.TTLGet()` to call it instead of `Get()`.

## Changes

### 1. Update ISysVvmStorage interface

File: `pkg/vvm/storage/interface.go`

Add `TTLGet` method:

```go
type ISysVvmStorage interface {
    InsertIfNotExists(pKey []byte, cCols []byte, value []byte, ttlSeconds int) (ok bool, err error)
    CompareAndSwap(pKey []byte, cCols []byte, oldValue, newValue []byte, ttlSeconds int) (ok bool, err error)
    CompareAndDelete(pKey []byte, cCols []byte, expectedValue []byte) (ok bool, err error)
    Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error)
    TTLGet(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error)  // NEW
    Put(pKey []byte, cCols []byte, value []byte) (err error)
    PutBatch(batch []istorage.BatchItem) error
}
```

### 2. Update implAppTTLStorage.TTLGet

File: `pkg/vvm/storage/impl_appttl.go`

Change from calling `Get()` to calling `TTLGet()`:

```go
func (s *implAppTTLStorage) TTLGet(key string) (value string, ok bool, err error) {
    if err := s.validateKey(key); err != nil {
        return "", false, err
    }
    pKey, cCols := s.buildKeys(key)
    var data []byte
    ok, err = s.sysVVMStorage.TTLGet(pKey, cCols, &data)  // Changed from Get to TTLGet
    if err != nil || !ok {
        return "", ok, err
    }
    return string(data), true, nil
}
```

## Notes

- The underlying `IAppStorage` interface already has `TTLGet()` method
- The cache layer (`cachedAppStorage`) already implements `TTLGet()` with proper expiration checking
- This change ensures TTL expiration is properly checked at the cache layer
