# Implementation plan: Use PER IP rate limit scope

## Construction

### Fix limiter `ResetLimits` to handle `RateScope_IP`

- [x] update: [pkg/appparts/internal/limiter/limiter.go](../../../../../pkg/appparts/internal/limiter/limiter.go)
  - update: Add `remoteAddr string` parameter to `ResetLimits` and set `key.RemoteAddr` when `RateScope_IP`, matching `Exceeded` logic
- [x] update: [pkg/appparts/internal/limiter/example_test.go](../../../../../pkg/appparts/internal/limiter/example_test.go)
  - update: Pass `remoteAddr` to `ResetLimits` in `Example_resetLimits`

### Update `IAppPartition.ResetRateLimit` signature

- [x] update: [pkg/appparts/interface.go](../../../../../pkg/appparts/interface.go)
  - update: Add `remoteAddr string` parameter to `ResetRateLimit` in `IAppPartition`
- [x] update: [pkg/appparts/impl_app.go](../../../../../pkg/appparts/impl_app.go)
  - update: Pass `remoteAddr` from `borrowedPartition.ResetRateLimit` to `limiter.ResetLimits`

### Populate `bus.Request.Host` with client IP in router

- [x] update: [pkg/bus/types.go](../../../../../pkg/bus/types.go)
  - update: `Host` field comment to describe it as client IP address (host only, port stripped)
- [x] update: [pkg/router/utils.go](../../../../../pkg/router/utils.go)
  - update: Set `bus.Request.Host` from `remoteIP(req.RemoteAddr)` in `createBusRequest`
  - add: `remoteIP` helper function using `net.SplitHostPort` to strip port
  - add: Host logging attribute in `withLogAttribs`

### Use `Host()` for rate limiting in processors

- [x] update: [pkg/processors/command/impl.go](../../../../../pkg/processors/command/impl.go)
  - update: `limitCallRate` to pass `cmd.cmdMes.Host()` instead of empty string
- [x] update: [pkg/processors/query/impl.go](../../../../../pkg/processors/query/impl.go)
  - update: Rate limit check to pass `qw.msg.Host()` instead of empty string
  - update: Simplify `ResetRateLimit` to derive WSID and Host internally from `qw.msg`
- [x] update: [pkg/processors/query2/util.go](../../../../../pkg/processors/query2/util.go)
  - update: `queryRateLimitExceeded` to pass `qw.msg.Host()` instead of empty string
  - update: Simplify `ResetRateLimit` to derive WSID and Host internally from `qw.msg`

### Update `ResetRateLimit` callers

- [x] update: [pkg/sys/verifier/impl.go](../../../../../pkg/sys/verifier/impl.go)
  - update: Simplify anonymous `ResetRateLimit` interface cast and call to `(appdef.QName, appdef.OperationKind)`

### Update mocks

- [x] update: [pkg/processors/schedulers/impl_test.go](../../../../../pkg/processors/schedulers/impl_test.go)
  - update: Mock `ResetRateLimit` signature to include `remoteAddr string`
- [x] Review

### Integration test for PER IP rate limit

- [x] update: [pkg/vit/schemaTestApp1.vsql](../../../../../pkg/vit/schemaTestApp1.vsql)
  - add: `IPRatedCmd` command and `IPRatedQry` query declarations
  - add: `RATE IPRatedPerMinute 2 PER MINUTE PER IP` and corresponding `LIMIT` declarations
- [x] update: [pkg/vit/shared_cfgs.go](../../../../../pkg/vit/shared_cfgs.go)
  - add: `QNameCmdIPRated` and `QNameQryIPRated` QName vars
  - add: Builtin function registrations for `IPRatedCmd` and `IPRatedQry`
- [x] update: [pkg/sys/it/impl_rates_test.go](../../../../../pkg/sys/it/impl_rates_test.go)
  - add: `TestRates_PerIP` — verifies PER IP rate limits using `127.0.0.1` (localhost test requests)
- [x] Review