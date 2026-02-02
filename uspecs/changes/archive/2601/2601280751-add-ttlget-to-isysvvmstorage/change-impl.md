# Implementation plan: Add TTLGet to ISysVvmStorage

## Construction

- [x] update: [pkg/vvm/storage/interface.go](../../../pkg/vvm/storage/interface.go)
  - Add `TTLGet(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error)` method to `ISysVvmStorage` interface

- [x] update: [pkg/vvm/storage/impl_appttl.go](../../../pkg/vvm/storage/impl_appttl.go)
  - Change `implAppTTLStorage.TTLGet()` to call `s.sysVVMStorage.TTLGet()` instead of `s.sysVVMStorage.Get()`

- [x] review
