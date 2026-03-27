# Decisions: Implement rate limits using VSQL

## Scope

"Preserve existing integration tests without modifications" means: do not modify tests in the `sys_it` package. Test application VSQL schemas and Go code are free to update. Replace Go-configured `FunctionRateLimits` calls with VSQL declarations and switch processors to use `appPart.IsLimitExceeded`. Legacy infrastructure elimination (`istructsmem` types, `bucketsFactory`, etc.) will be done in a separate pull request.

## Replace appconfig rate limits with VSQL declarations

All rate limits configured via `cfg.FunctionRateLimits.AddAppLimit` / `cfg.FunctionRateLimits.AddWorkspaceLimit` must be replaced with VSQL `RATE` and `LIMIT` declarations (confidence: high).

Affected callers:

- `pkg/vit/shared_cfgs.go` — test app `RatedCmd`/`RatedQry` limits
- `pkg/sys/verifier/provide.go` — `ProvideLimits()` for `InitiateEmailVerification` and `IssueVerifiedValueToken`
- `pkg/registry/impl_changepassword.go` — `provideChangePassword()` for `CmdChangePassword`

Rationale: The VSQL parser already supports `RATE` and `LIMIT` syntax. The appparts limiter reads limits from appdef populated by VSQL. Declaring limits in the schema eliminates the need for manual Go-based setup.

Alternatives:

- Keep Go-based limits alongside VSQL declarations (confidence: low)
  - Results in double rate limiting, contradicts the goal
- Migrate test app only, keep verifier/registry on old system (confidence: low)
  - Leaves dead infrastructure in place, contradicts the goal of full elimination

## VSQL RATE scope mapping for per-app limits

The old system supports `RateLimitKind_byApp` (per-app limits keyed by `AppQName` — single bucket for the entire app). VSQL RATE supports scopes: `PER APP PARTITION`, `PER WORKSPACE`, `PER IP`, `PER SUBJECT`. There is no direct `PER APP` scope.

A new `PER APP` scope is NOT required. Use `PER APP PARTITION` as the replacement (confidence: high).

Rationale: In the old system, `RateLimitKind_byApp` creates one bucket keyed by `AppQName`. With `PER APP PARTITION`, each partition has its own `Limiter` instance with its own `IBuckets`, so limits are per-partition. However, the integration test (`TestRates_BasicUsage`) sends all requests to a single workspace which maps to a single partition. Therefore the behavior is identical whether the limit is per-app or per-partition. The new system's per-partition design is intentional and provides better scalability.

Alternatives:

- Implement a new `RateScope_App` in `appdef` and `limiter` (confidence: low)
  - Requires parser, appdef, and limiter changes; unnecessary since the integration test behavior is preserved with `PER APP PARTITION`
- Use `PER WORKSPACE` instead (confidence: low)
  - Different semantics, limits would be per-workspace not per-partition

## Processor-level rate limit check: `IsLimitExceeded` vs `IsFunctionRateLimitsExceeded`

Two rate limit check methods exist:

- `IAppPartition.IsLimitExceeded(resource, operation, workspace, remoteAddr)` — delegates to `limiter.Exceeded()`, reads limits from `appdef.IAppDef` (VSQL RATE/LIMIT declarations), per-partition buckets, supports scopes: `PER APP PARTITION`, `PER WORKSPACE`, `PER IP`
- `IAppStructs.IsFunctionRateLimitsExceeded(funcQName, wsid)` — reads limits from `config.FunctionRateLimits` (Go-configured via `cfg.FunctionRateLimits.AddAppLimit`/`AddWorkspaceLimit`), app-level buckets, supports scopes: `byApp`, `byWorkspace`

Use `appPart.IsLimitExceeded`, eliminate `IsFunctionRateLimitsExceeded` from processors (confidence: high).

Rationale:

- `IsFunctionRateLimitsExceeded` is already marked `FIXME: eliminate` in `IAppStructs` interface
- `IsLimitExceeded` reads limits from `appdef.IAppDef` populated by VSQL — aligns with the goal of VSQL-based rate limits
- `IsLimitExceeded` supports `remoteAddr` for `PER IP` scope — richer than the old method
- `IsFunctionRateLimitsExceeded` depends on `config.FunctionRateLimits` which we are removing
- `IsLimitExceeded` lives on `IAppPartition` (borrowed partition) which is already available in all three processors

Alternatives:

- Keep `IsFunctionRateLimitsExceeded` (confidence: low)
  - Marked for elimination, reads from Go-configured limits only, cannot read VSQL declarations

## Verifier bucket reset

The verifier in `pkg/sys/verifier/impl.go` resets rate limit buckets on successful verification. With the switch to partition-level limiter, the reset must target the partition's `Limiter` instance via `IAppPartition.ResetRateLimit`.

Use a workpiece-based approach: the query processor's workpiece implements an anonymous `ResetRateLimit` interface, and the verifier casts the workpiece to call it (confidence: high).

Rationale: The verifier runs as a query function and does not have direct access to `IAppPartition`. The workpiece-based cast avoids adding `IAppPartition` as a dependency to the verifier while preserving the reset behavior.

Alternatives:

- Remove the bucket reset entirely — let rate limits expire naturally (confidence: low)
  - Changes observable behavior for users who retry immediately after successful verification
- Expose partition buckets via state/intents mechanism (confidence: low)
  - Over-engineered for a single use case

## VIT `MockBuckets` elimination

Done in [AIR-3414](https://untill.atlassian.net/browse/AIR-3414) (change `2603261339-elim-mock-buckets`): `MockBuckets` was removed from `pkg/vit/impl.go` and integration tests in `sys_it` were updated to use natural bucket depletion.

## Affected tests

Tests that use `cfg.FunctionRateLimits` for rate limit setup must be migrated to VSQL-based `wsb.AddRate`/`wsb.AddLimit`:

- `pkg/istructsmem/impl_test.go` — replace `cfg.FunctionRateLimits` with `wsb.AddRate`/`wsb.AddLimit`
- `pkg/processors/command/impl_test.go` `TestRateLimit` — migrate to VSQL-based rate limit setup
- `pkg/processors/query/impl_test.go` — migrate to VSQL-based rate limit setup
