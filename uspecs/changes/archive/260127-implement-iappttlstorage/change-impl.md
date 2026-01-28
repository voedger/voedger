# Implementation plan: Implement IAppTTLStorage interface

## Technical design

- [x] create: [storage/appttl--arch.md](../../specs/prod/storage/appttl--arch.md)
  - Document Application TTL Storage subsystem architecture
  - Define IAppTTLStorage interface placement and relationship with ISysVvmStorage
  - Describe component hierarchy and data flow
  - Define validation rules for key, value, and TTL parameters

## Construction

- [x] update: [pkg/istructs/interface.go](../../../pkg/istructs/interface.go)
  - Add `IAppTTLStorage` interface with TTLGet, InsertIfNotExists, CompareAndSwap, CompareAndDelete methods
  - Add `AppTTLStorage() IAppTTLStorage` method to `IAppStructs` interface
  - Add `AppTTLStorageFactory` function type
- [x] fix: [pkg/istructs/interface.go](../../../pkg/istructs/interface.go)
  - Remove `ISysVvmStorage` interface (belongs to `pkg/vvm/storage` only)
- [x] update: [pkg/vvm/storage/consts.go](../../../pkg/vvm/storage/consts.go)
  - Add `pKeyPrefix_AppTTL` constant (value 4)
  - Add validation constants: `MaxKeyLength`, `MaxValueLength`, `MaxTTLSeconds`
- [x] create: [pkg/vvm/storage/errors.go](../../../pkg/vvm/storage/errors.go)
  - Define validation errors: `ErrKeyEmpty`, `ErrKeyTooLong`, `ErrValueTooLong`, `ErrInvalidTTL`
- [x] create: [pkg/vvm/storage/impl_appttl.go](../../../pkg/vvm/storage/impl_appttl.go)
  - Implement `implAppTTLStorage` struct wrapping `ISysVvmStorage` from local package
  - Implement buildKeys, validation methods, and all IAppTTLStorage methods
- [x] update: [pkg/vvm/storage/provide.go](../../../pkg/vvm/storage/provide.go)
  - Add `NewAppTTLStorage` factory function
- [x] update: [pkg/istructsmem/provide.go](../../../pkg/istructsmem/provide.go)
  - Add `appTTLStorageFactory` parameter to `Provide` function
- [x] update: [pkg/istructsmem/impl.go](../../../pkg/istructsmem/impl.go)
  - Add `appTTLStorageFactory` field to `appStructsProviderType`
  - Add `appTTLStorage` field to `appStructsType`
  - Implement `AppTTLStorage()` method
- [x] update: [pkg/vvm/provide.go](../../../pkg/vvm/provide.go)
  - Update `provideIAppStructsProvider` to create factory capturing `sysVvmStorage`
- [x] regenerate: [pkg/vvm/wire_gen.go](../../../pkg/vvm/wire_gen.go)
  - Run Wire to regenerate dependency injection
- [x] create: [pkg/vvm/storage/impl_appttl_test.go](../../../pkg/vvm/storage/impl_appttl_test.go)
  - Test validation errors and operations with expiration using `testingu.MockTime.Add()`
- [x] update: test files calling `istructsmem.Provide()`
  - Add `nil` parameter for `appTTLStorageFactory`
- [x] review
