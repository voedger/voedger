# Implementation plan: Eliminate AppConfig rate limits infrastructure

## Construction

### Interface and type removal

- [x] update: [pkg/istructs/interface.go](../../../../pkg/istructs/interface.go)
  - remove: `IsFunctionRateLimitsExceeded` method from `IAppStructs` interface
- [x] update: [pkg/istructs/recources-types.go](../../../../pkg/istructs/recources-types.go)
  - remove: `RateLimitKind` type and `RateLimit` struct
- [x] update: [pkg/istructs/consts.go](../../../../pkg/istructs/consts.go)
  - remove: `RateLimitKind_byApp`, `RateLimitKind_byWorkspace`, `RateLimitKind_byID`, `RateLimitKind_FakeLast` constants
- [x] delete: [pkg/istructs/ratelimitkind_string.go](../../../../pkg/istructs/ratelimitkind_string.go)
  - remove: generated stringer for `RateLimitKind`
- [x] update: [pkg/istructs/utils_test.go](../../../../pkg/istructs/utils_test.go)
  - remove: `TestRateLimitKind_String` and `TestRateLimitKind_MarshalText` tests
- [x] update: [pkg/istructs/utils.go](../../../../pkg/istructs/utils.go)
  - remove: `RateLimitKind.MarshalText` method

- [x] Review

### istructsmem rate limit infrastructure removal

- [x] delete: [pkg/istructsmem/ratelimits-types.go](../../../../pkg/istructsmem/ratelimits-types.go)
  - remove: `functionRateLimits` struct with `AddAppLimit`, `AddWorkspaceLimit`, `prepare`, `addFuncLimit` methods and `GetFunctionRateLimitName` function
- [x] delete: [pkg/istructsmem/ratelimits-types_test.go](../../../../pkg/istructsmem/ratelimits-types_test.go)
  - remove: `TestRateLimits_BasicUsage`, `TestRateLimitsErrors`, `TestGetFunctionRateLimitName` tests
- [x] update: [pkg/istructsmem/consts.go](../../../../pkg/istructsmem/consts.go)
  - remove: `funcRateLimitNameFmt` variable
- [x] update: [pkg/istructsmem/appstruct-types.go](../../../../pkg/istructsmem/appstruct-types.go)
  - remove: `FunctionRateLimits` field from `AppConfigType`
  - remove: `FunctionRateLimits` initialization in `newAppConfig`
  - update: `prepare` method — remove `buckets irates.IBuckets` parameter and `cfg.FunctionRateLimits.prepare(buckets)` call
- [x] update: [pkg/istructsmem/impl.go](../../../../pkg/istructsmem/impl.go)
  - remove: `IsFunctionRateLimitsExceeded` method from `appStructsType`
  - remove: `buckets` field and `Buckets()` method from `appStructsType`
  - remove: `bucketsFactory` field from `appStructsProviderType`
  - update: `newAppStructs` — remove `buckets` parameter
  - update: `BuiltIn` and `New` methods — remove `bucketsFactory()` call and `buckets` argument from `prepare` and `newAppStructs`
- [x] update: [pkg/istructsmem/provide.go](../../../../pkg/istructsmem/provide.go)
  - remove: `bucketsFactory` parameter from `Provide` function
- [x] update: [pkg/istructsmem/utils.go](../../../../pkg/istructsmem/utils.go)
  - remove: `IBucketsFromIAppStructs` function
- [x] update: [pkg/istructsmem/utils_test.go](../../../../pkg/istructsmem/utils_test.go)
  - remove: `TestIBucketsFromIAppStructs` test

- [x] Review

### Caller updates

- [x] update: [pkg/iauthnzimpl/impl_test.go](../../../../pkg/iauthnzimpl/impl_test.go)
  - remove: `IsFunctionRateLimitsExceeded` from `implIAppStructs` mock
- [x] update: [pkg/vvm/provide.go](../../../../pkg/vvm/provide.go)
  - update: `provideIAppStructsProvider` — remove `bucketsFactory` parameter and its forwarding to `istructsmem.Provide`
- [x] update: [pkg/vvm/wire_gen.go](../../../../pkg/vvm/wire_gen.go)
  - update: `provideIAppStructsProvider` call — remove `bucketsFactoryType` argument
  - update: `provideIAppStructsProvider` function — remove `bucketsFactory` parameter
- [x] update: [pkg/appparts/example_test.go](../../../../pkg/appparts/example_test.go)
  - update: `istructsmem.Provide` call — remove buckets factory argument
- [x] update: [pkg/appparts/example_limit_test.go](../../../../pkg/appparts/example_limit_test.go)
  - update: `istructsmem.Provide` call — remove buckets factory argument
- [x] update: [pkg/appparts/impl_test.go](../../../../pkg/appparts/impl_test.go)
  - update: `istructsmem.Provide` call — remove buckets factory argument
- [x] update: [pkg/iextengine/wazero/impl_test.go](../../../../pkg/iextengine/wazero/impl_test.go)
  - update: `istructsmem.Provide` call — remove buckets factory argument
- [x] update: [pkg/parser/impl_test.go](../../../../pkg/parser/impl_test.go)
  - update: `istructsmem.Provide` calls — remove buckets factory argument
  - update: `appparts.New2` calls — replace `irates.NullBucketsFactory` with `nil`
- [x] update: [pkg/processors/actualizers/impl_test.go](../../../../pkg/processors/actualizers/impl_test.go)
  - update: `istructsmem.Provide` call — remove buckets factory argument
- [x] update: [pkg/processors/command/impl_test.go](../../../../pkg/processors/command/impl_test.go)
  - update: `istructsmem.Provide` call — remove buckets factory argument
- [x] update: [pkg/processors/query/impl_test.go](../../../../pkg/processors/query/impl_test.go)
  - update: `istructsmem.Provide` call — remove buckets factory argument
- [x] update: [pkg/state/teststate/impl.go](../../../../pkg/state/teststate/impl.go)
  - update: `istructsmem.Provide` call — remove buckets factory argument
- [x] update: [pkg/state/teststate/impl_new.go](../../../../pkg/state/teststate/impl_new.go)
  - update: `istructsmem.Provide` call — remove buckets factory argument
- [x] update: [pkg/sys/collection/collection_test.go](../../../../pkg/sys/collection/collection_test.go)
  - update: `istructsmem.Provide` call — remove buckets factory argument
- [x] update: [pkg/sys/storages/impl_event_storage_test.go](../../../../pkg/sys/storages/impl_event_storage_test.go)
  - update: `istructsmem.Provide` call — remove buckets factory argument
- [x] update: [pkg/istructsmem/impl_test.go](../../../../pkg/istructsmem/impl_test.go)
  - update: `Provide` calls — remove buckets factory argument
- [x] update: [pkg/istructsmem/appstruct-types_test.go](../../../../pkg/istructsmem/appstruct-types_test.go)
  - update: `Provide` calls — remove buckets factory argument
- [x] update: [pkg/istructsmem/bench_test.go](../../../../pkg/istructsmem/bench_test.go)
  - update: `Provide` calls — remove buckets factory argument
- [x] update: [pkg/istructsmem/event-types_test.go](../../../../pkg/istructsmem/event-types_test.go)
  - update: `Provide` calls — remove buckets factory argument
- [x] update: [pkg/istructsmem/impl_seqtrust_test.go](../../../../pkg/istructsmem/impl_seqtrust_test.go)
  - update: `Provide` calls — remove buckets factory argument
- [x] update: [pkg/istructsmem/records-types_test.go](../../../../pkg/istructsmem/records-types_test.go)
  - update: `Provide` calls — remove buckets factory argument
- [x] update: [pkg/istructsmem/resources-types_test.go](../../../../pkg/istructsmem/resources-types_test.go)
  - update: `Provide` calls — remove buckets factory argument
- [x] update: [pkg/istructsmem/test_test.go](../../../../pkg/istructsmem/test_test.go)
  - update: `Provide` calls — remove buckets factory argument
- [x] update: [pkg/istructsmem/validation_test.go](../../../../pkg/istructsmem/validation_test.go)
  - update: `Provide` calls — remove buckets factory argument
- [x] update: [pkg/istructsmem/viewrecords-types_test.go](../../../../pkg/istructsmem/viewrecords-types_test.go)
  - update: `Provide` calls — remove buckets factory argument

- [x] Review

### NullBucketsFactory and NullBucket elimination

- [x] delete: [pkg/irates/consts.go](../../../../pkg/irates/consts.go)
  - remove: `NullBucketsFactory` variable and `NullBucket` type with all methods
- [x] update: [pkg/appparts/provide.go](../../../../pkg/appparts/provide.go)
  - remove: `New` function (simplified test constructor)
- [x] update: [pkg/appparts/test_utils.go](../../../../pkg/appparts/test_utils.go)
  - update: replace `irates.NullBucketsFactory` with `iratesce.TestBucketsFactory`
- [x] update: [pkg/parser/impl_test.go](../../../../pkg/parser/impl_test.go)
  - update: `appparts.New2` calls — replace `irates.NullBucketsFactory` with `nil`
  - remove: `irates` import
- [x] Review
