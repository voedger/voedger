# How: Implement rate limits using VSQL

## Approach

### 1. Add VSQL RATE/LIMIT declarations

- `pkg/vit/schemaTestApp1.vsql` — add RATE and LIMIT for `RatedCmd`/`RatedQry`: 2 per minute (`PER APP PARTITION`), 4 per hour (`PER WORKSPACE`), using `EACH` filter option
- `pkg/sys/verifier/schema.vsql` or sys package VSQL — add RATE and LIMIT for `InitiateEmailVerification` and `IssueVerifiedValueToken` (workspace-scoped)
- `pkg/registry/schema.vsql` or registry VSQL — add RATE and LIMIT for `CmdChangePassword` (partition-scoped, 1 per minute)

### 2. Remove Go-configured rate limits

- `pkg/vit/shared_cfgs.go` — remove `cfg.FunctionRateLimits.AddAppLimit` and `AddWorkspaceLimit` calls for `RatedCmd`/`RatedQry`
- `pkg/sys/verifier/provide.go` — remove `ProvideLimits()` function or its body
- `pkg/registry/impl_changepassword.go` — remove `cfg.FunctionRateLimits.AddAppLimit` call
- `pkg/sys/verifier/consts.go` — remove `RateLimit_IssueVerifiedValueToken` variable (uses `istructs.RateLimit` type)

### 3. Swap processor rate limit check

Replace `appStructs.IsFunctionRateLimitsExceeded(qName, wsid)` with `appPart.IsLimitExceeded(qName, operationKind, wsid, remoteAddr)` in:

- `pkg/processors/command/impl.go` — `limitCallRate` function
- `pkg/processors/query/impl.go` — inline rate check operator
- `pkg/processors/query2/util.go` — `queryRateLimitExceeded` function

### 4. Switch verifier bucket reset to partition-level limiter

- `pkg/appparts/internal/limiter/limiter.go` — add `ResetLimits` method
- `pkg/appparts/interface.go` — add `ResetRateLimit` to `IAppPartition`
- `pkg/appparts/impl_app.go` — implement `ResetRateLimit` on `borrowedPartition`
- `pkg/processors/query/impl.go` — add `ResetRateLimit` on `queryWork` workpiece
- `pkg/processors/query2/util.go` — add `ResetRateLimit` on `queryWork` workpiece
- `pkg/sys/verifier/impl.go` — replace `IBucketsFromIAppStructs(as).ResetRateBuckets(...)` with workpiece cast to anonymous `ResetRateLimit` interface

### 5. Update affected tests

- `pkg/istructsmem/impl_test.go` — replace `cfg.FunctionRateLimits` with `wsb.AddRate`/`wsb.AddLimit`
- `pkg/processors/command/impl_test.go` — `TestRateLimit`: migrate to VSQL-based rate limit setup
- `pkg/processors/query/impl_test.go` — migrate rate limit test to VSQL-based setup

## References

- [pkg/vit/schemaTestApp1.vsql](../../../../pkg/vit/schemaTestApp1.vsql)
- [pkg/vit/shared_cfgs.go](../../../../pkg/vit/shared_cfgs.go)
- [pkg/vit/consts.go](../../../../pkg/vit/consts.go)
- [pkg/vit/impl.go](../../../../pkg/vit/impl.go)
- [pkg/processors/command/impl.go](../../../../pkg/processors/command/impl.go)
- [pkg/processors/query/impl.go](../../../../pkg/processors/query/impl.go)
- [pkg/processors/query2/util.go](../../../../pkg/processors/query2/util.go)
- [pkg/appparts/interface.go](../../../../pkg/appparts/interface.go)
- [pkg/appparts/internal/limiter/limiter.go](../../../../pkg/appparts/internal/limiter/limiter.go)
- [pkg/appparts/internal/limiter/limiter.go](../../../../pkg/appparts/internal/limiter/limiter.go)
- [pkg/appparts/impl_app.go](../../../../pkg/appparts/impl_app.go)
- [pkg/sys/verifier/provide.go](../../../../pkg/sys/verifier/provide.go)
- [pkg/sys/verifier/impl.go](../../../../pkg/sys/verifier/impl.go)
- [pkg/sys/verifier/consts.go](../../../../pkg/sys/verifier/consts.go)
- [pkg/registry/impl_changepassword.go](../../../../pkg/registry/impl_changepassword.go)
- [pkg/sys/it/impl_verifier_test.go](../../../../pkg/sys/it/impl_verifier_test.go)
- [pkg/sys/it/impl_resetpassword_test.go](../../../../pkg/sys/it/impl_resetpassword_test.go)
