---
registered_at: 2026-01-27T14:56:27Z
change_id: 260127-implement-appttl-integration-tests
baseline: 131ee4b938a06cad88c47c9b9bec590457bb17cf
archived_at: 2026-01-28T08:18:32Z
---

# Change request: Implement integration tests for App TTL Storage

## Why

IAppTTLStorage implementation needs integration tests that verify the feature works correctly from HTTP client perspective. Unit tests exist but integration tests are needed to ensure the full stack works correctly including command/query handlers, HTTP routing, and TTL expiration.

## What

Add integration tests for App TTL Storage in `pkg/sys/it` package:

1. Add VSQL definitions in `pkg/vit/schemaTestApp1.vsql`:
   - `TTLStorageParams` type with fields: Operation (text), Key (text), Value (text), ExpectedValue (text), TTLSeconds (int32)
   - `TTLStorageResult` type with fields: Ok (bool)
   - `TTLStorageCmd` command using TTLStorageParams, returns TTLStorageResult
   - `TTLGetParams` type with fields: Key (text)
   - `TTLGetResult` type with fields: Value (text), Exists (bool)
   - `TTLGetQry` query using TTLGetParams, returns TTLGetResult

2. Add Go implementations in `pkg/vit/shared_cfgs.go` within `ProvideApp1` function:
   - `c.app1pkg.TTLStorageCmd` - handles operations:
     - "Put" - calls `InsertIfNotExists(key, value, ttlSeconds)`
     - "CompareAndSwap" - calls `CompareAndSwap(key, expectedValue, value, ttlSeconds)`
     - "CompareAndDelete" - calls `CompareAndDelete(key, expectedValue)`
   - `q.app1pkg.TTLGetQry` - calls `TTLGet(key)`, returns value and exists flag

3. Access IAppTTLStorage via:

   ```go
   ttlStorage := args.State.AppStructs().AppTTLStorage()
   ```

4. Error handling in command and query implementations:
   - Check if error is `storage.ErrAppTTLValidation` using `errors.Is()`
   - If validation error: return `coreutils.WrapSysError(err, http.StatusBadRequest)` → HTTP 400
   - Otherwise: return error as-is → HTTP 500

   ```go
   if err != nil {
       if errors.Is(err, storage.ErrAppTTLValidation) {
           return coreutils.WrapSysError(err, http.StatusBadRequest)
       }
       return err
   }
   ```

5. Create integration test file `pkg/sys/it/impl_appttl_test.go` with tests:
   - Test basic Put and Get operations
   - Test TTL expiration using `vit.TimeAdd(duration)`
   - Test CompareAndSwap with correct and wrong expected values
   - Test CompareAndDelete with correct and wrong expected values
   - Test InsertIfNotExists when key already exists
