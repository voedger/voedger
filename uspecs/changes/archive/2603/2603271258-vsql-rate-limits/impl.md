# Implementation plan: Implement rate limits using VSQL

## Construction

### VSQL declarations

- [x] update: [pkg/vit/schemaTestApp1.vsql](../../../../pkg/vit/schemaTestApp1.vsql)
  - add: RATE and LIMIT for `RatedCmd`/`RatedQry` — 2 per minute `PER APP PARTITION` with `EACH`, 4 per hour `PER WORKSPACE` with `EACH`
- [x] update: [pkg/sys/userprofile.vsql](../../../../pkg/sys/userprofile.vsql)
  - add: RATE and LIMIT for `InitiateEmailVerification` and `IssueVerifiedValueToken` — workspace-scoped, matching legacy names via `GetFunctionRateLimitName` format
- [x] update: [pkg/registry/appws.vsql](../../../../pkg/registry/appws.vsql)
  - add: RATE and LIMIT for `ChangePassword` — partition-scoped, 1 per minute
- [x] Review

### Remove Go-configured rate limits

- [x] update: [pkg/vit/shared_cfgs.go](../../../../pkg/vit/shared_cfgs.go)
  - remove: `cfg.FunctionRateLimits.AddAppLimit` and `AddWorkspaceLimit` calls for `RatedCmd`/`RatedQry`
- [x] update: [pkg/sys/verifier/provide.go](../../../../pkg/sys/verifier/provide.go)
  - remove: `ProvideLimits()` function or its rate-limit-related body
- [x] update: [pkg/registry/impl_changepassword.go](../../../../pkg/registry/impl_changepassword.go)
  - remove: `cfg.FunctionRateLimits.AddAppLimit` call for `CmdChangePassword`
- [x] update: [pkg/sys/verifier/consts.go](../../../../pkg/sys/verifier/consts.go)
  - remove: `RateLimit_IssueVerifiedValueToken` variable
- [x] Review

### Swap processor rate limit check

- [x] update: [pkg/processors/command/impl.go](../../../../pkg/processors/command/impl.go)
  - update: `limitCallRate` to use `appPart.IsLimitExceeded` instead of `appStructs.IsFunctionRateLimitsExceeded`
- [x] update: [pkg/processors/query/impl.go](../../../../pkg/processors/query/impl.go)
  - update: inline rate check to use `appPart.IsLimitExceeded`
- [x] update: [pkg/processors/query2/util.go](../../../../pkg/processors/query2/util.go)
  - update: `queryRateLimitExceeded` to use `appPart.IsLimitExceeded`
- [x] check: integration tests that check rate limits for `RatedCmd`/`RatedQry`, `InitiateEmailVerification`, `IssueVerifiedValueToken`, and `ChangePassword` shall pass
- [x] Review

### Switch verifier bucket reset to partition-level limiter

- [x] update: [pkg/appparts/internal/limiter/limiter.go](../../../../pkg/appparts/internal/limiter/limiter.go)
  - add: `ResetLimits` method
- [x] update: [pkg/appparts/internal/limiter/example_test.go](../../../../pkg/appparts/internal/limiter/example_test.go)
  - add: `Example_resetLimits` showing `Limiter.ResetLimits` usage
- [x] update: [pkg/appparts/interface.go](../../../../pkg/appparts/interface.go)
  - add: `ResetRateLimit` to `IAppPartition`
- [x] update: [pkg/appparts/impl_app.go](../../../../pkg/appparts/impl_app.go)
  - add: `ResetRateLimit` on `borrowedPartition` delegating to limiter
- [x] update: [pkg/processors/query/impl.go](../../../../pkg/processors/query/impl.go)
  - add: `ResetRateLimit` method on `queryWork` delegating to `appPart.ResetRateLimit`
- [x] update: [pkg/processors/query2/util.go](../../../../pkg/processors/query2/util.go)
  - add: `ResetRateLimit` method on `queryWork` delegating to `appPart.ResetRateLimit`
- [x] update: [pkg/sys/verifier/impl.go](../../../../pkg/sys/verifier/impl.go)
  - replace: `IBucketsFromIAppStructs(as).ResetRateBuckets(...)` with workpiece cast to anonymous `ResetRateLimit` interface
- [x] update: [pkg/processors/schedulers/impl_test.go](../../../../pkg/processors/schedulers/impl_test.go)
  - add: `ResetRateLimit` to mock `IAppPartition`
- [x] Review

### Update affected tests

- [x] update: [pkg/istructsmem/impl_test.go](../../../../pkg/istructsmem/impl_test.go)
  - replace: `cfg.FunctionRateLimits.AddAppLimit`/`AddWorkspaceLimit` with `wsb.AddRate`/`wsb.AddLimit` in `Test_BasicUsageDescribePackages`
- [x] update: [pkg/processors/command/impl_test.go](../../../../pkg/processors/command/impl_test.go)
  - replace: `cfg.FunctionRateLimits.AddWorkspaceLimit` with `wsb.AddRate`/`wsb.AddLimit` in `TestRateLimit`
- [x] update: [pkg/processors/query/impl_test.go](../../../../pkg/processors/query/impl_test.go)
  - move: rate limit setup from `cfg` callback to `wsb` callback using `wsb.AddRate`/`wsb.AddLimit` in `TestRateLimiter`
- [x] Review
