# How: Eliminate AppConfig rate limits infrastructure

## Approach

- Remove `FunctionRateLimits` field from `AppConfigType` in `pkg/istructsmem/appstruct-types.go` and its initialization in `newAppConfig`
- Delete `pkg/istructsmem/ratelimits-types.go` (contains `functionRateLimits` struct and all its methods: `AddAppLimit`, `AddWorkspaceLimit`, `prepare`, `addFuncLimit`)
- Delete `pkg/istructsmem/ratelimits-types_test.go` (tests for the removed types)
- Remove `funcRateLimitNameFmt` variable and `GetFunctionRateLimitName` function from `pkg/istructsmem/consts.go`
- Remove `IsFunctionRateLimitsExceeded` method from `appStructsType` in `pkg/istructsmem/impl.go`
- Remove `IsFunctionRateLimitsExceeded` from `IAppStructs` interface in `pkg/istructs/interface.go`
- Remove `RateLimitKind` type, `RateLimit` struct, and related constants (`RateLimitKind_byApp`, `RateLimitKind_byWorkspace`, `RateLimitKind_byID`, `RateLimitKind_FakeLast`) from `pkg/istructs/consts.go` and `pkg/istructs/recources-types.go`
- Delete generated stringer file `pkg/istructs/ratelimitkind_string.go`
- Remove `RateLimitKind` tests from `pkg/istructs/utils_test.go` (`TestRateLimitKind_String`, `TestRateLimitKind_MarshalText`)
- Remove `buckets` field and `Buckets()` method from `appStructsType` and `bucketsFactory` from `appStructsProviderType` in `pkg/istructsmem/impl.go`
- Remove `IBucketsFromIAppStructs` from `pkg/istructsmem/utils.go` (no production callers remain; only used in its own test)
- Remove `bucketsFactory` parameter from `Provide` function in `pkg/istructsmem/provide.go` and update all callers
- Remove `cfg.FunctionRateLimits.prepare(buckets, ...)` call from the config prepare path
- Remove `TestIBucketsFromIAppStructs` test from `pkg/istructsmem/utils_test.go`
- Clean up any remaining test references that use `FunctionRateLimits`

References:

- [pkg/istructsmem/appstruct-types.go](../../../../pkg/istructsmem/appstruct-types.go)
- [pkg/istructsmem/ratelimits-types.go](../../../../pkg/istructsmem/ratelimits-types.go)
- [pkg/istructsmem/ratelimits-types_test.go](../../../../pkg/istructsmem/ratelimits-types_test.go)
- [pkg/istructsmem/consts.go](../../../../pkg/istructsmem/consts.go)
- [pkg/istructsmem/impl.go](../../../../pkg/istructsmem/impl.go)
- [pkg/istructsmem/provide.go](../../../../pkg/istructsmem/provide.go)
- [pkg/istructs/interface.go](../../../../pkg/istructs/interface.go)
- [pkg/istructs/consts.go](../../../../pkg/istructs/consts.go)
- [pkg/istructs/recources-types.go](../../../../pkg/istructs/recources-types.go)
- [pkg/istructs/ratelimitkind_string.go](../../../../pkg/istructs/ratelimitkind_string.go)
- [pkg/istructs/utils_test.go](../../../../pkg/istructs/utils_test.go)
- [pkg/istructsmem/utils.go](../../../../pkg/istructsmem/utils.go)
- [pkg/istructsmem/utils_test.go](../../../../pkg/istructsmem/utils_test.go)

