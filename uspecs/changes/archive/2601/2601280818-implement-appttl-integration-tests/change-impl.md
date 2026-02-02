# Implementation plan: Implement integration tests for App TTL Storage

## Construction

- [x] update: [pkg/vit/schemaTestApp1.vsql](../../../pkg/vit/schemaTestApp1.vsql)
  - Add `TTLStorageParams` type with fields: Operation (text), Key (text), Value (text), ExpectedValue (text), TTLSeconds (int32)
  - Add `TTLStorageResult` type with fields: Ok (bool)
  - Add `TTLStorageCmd` command using TTLStorageParams, returns TTLStorageResult
  - Add `TTLGetParams` type with fields: Key (text)
  - Add `TTLGetResult` type with fields: Value (text), Exists (bool)
  - Add `TTLGetQry` query using TTLGetParams, returns TTLGetResult
  - Add grants for sys.WorkspaceOwner

- [x] update: [pkg/vit/shared_cfgs.go](../../../pkg/vit/shared_cfgs.go)
  - Add command implementation for `TTLStorageCmd` handling Put/CompareAndSwap/CompareAndDelete operations
  - Add query implementation for `TTLGetQry` returning value and exists flag
  - Use `args.State.AppStructs().AppTTLStorage()` to access storage
  - Wrap validation errors with `coreutils.WrapSysError(err, http.StatusBadRequest)`

- [x] create: [pkg/sys/it/impl_appttl_test.go](../../../pkg/sys/it/impl_appttl_test.go)
  - Test basic Put and Get operations
  - Test TTL expiration using `vit.TimeAdd(duration)`
  - Test CompareAndSwap with correct and wrong expected values
  - Test CompareAndDelete with correct and wrong expected values
  - Test InsertIfNotExists when key already exists
  - Test validation errors return HTTP 400

- [x] review
