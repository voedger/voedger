# Implementation plan: Use PER IP rate limit scope

## Construction

### Fix limiter `ResetLimits` to handle `RateScope_IP`

- [x] update: [pkg/appparts/internal/limiter/limiter.go](../../../pkg/appparts/internal/limiter/limiter.go)
  - update: Add `remoteAddr string` parameter to `ResetLimits` and set `key.RemoteAddr` when `RateScope_IP`, matching `Exceeded` logic
- [x] update: [pkg/appparts/internal/limiter/example_test.go](../../../pkg/appparts/internal/limiter/example_test.go)
  - update: Pass `remoteAddr` to `ResetLimits` in `Example_resetLimits`

### Update `IAppPartition.ResetRateLimit` signature

- [x] update: [pkg/appparts/interface.go](../../../pkg/appparts/interface.go)
  - update: Add `remoteAddr string` parameter to `ResetRateLimit` in `IAppPartition`
- [x] update: [pkg/appparts/impl_app.go](../../../pkg/appparts/impl_app.go)
  - update: Pass `remoteAddr` from `borrowedPartition.ResetRateLimit` to `limiter.ResetLimits`

### Repurpose `bus.Request.Host` as client IP and propagate from router

- [x] update: [pkg/bus/types.go](../../../pkg/bus/types.go)
  - update: Remove `RemoteAddr` field, update `Host` comment to describe it as client IP address
- [x] update: [pkg/router/utils.go](../../../pkg/router/utils.go)
  - update: Set `bus.Request.Host` from `remoteIP(req.RemoteAddr)` in `createBusRequest`, stripping port via `net.SplitHostPort`
  - add: `remoteIP` helper function

### Use `Host()` for rate limiting in processors

- [x] update: [pkg/processors/command/types.go](../../../pkg/processors/command/types.go)
  - remove: `RemoteAddr() string` from `ICommandMessage` and `remoteAddr` field from `implICommandMessage`
- [x] update: [pkg/processors/command/impl.go](../../../pkg/processors/command/impl.go)
  - update: Remove `remoteAddr` parameter from `NewCommandMessage`
  - update: `limitCallRate` to pass `cmd.cmdMes.Host()` instead of `cmd.cmdMes.RemoteAddr()`
- [x] update: [pkg/processors/query/types.go](../../../pkg/processors/query/types.go)
  - remove: `RemoteAddr() string` from `IQueryMessage` interface
- [x] update: [pkg/processors/query/impl.go](../../../pkg/processors/query/impl.go)
  - remove: `remoteAddr` field from `queryMessage` struct and `RemoteAddr()` accessor
  - update: Remove `remoteAddr` parameter from `NewQueryMessage`
  - update: Rate limit check to pass `qw.msg.Host()` instead of `qw.msg.RemoteAddr()`
- [x] update: [pkg/processors/query2/types.go](../../../pkg/processors/query2/types.go)
  - remove: `RemoteAddr() string` from `IQueryMessage` interface and `remoteAddr` field from `implIQueryMessage`
- [x] update: [pkg/processors/query2/util.go](../../../pkg/processors/query2/util.go)
  - update: Remove `remoteAddr` parameter from `NewIQueryMessage`
  - update: `queryRateLimitExceeded` to pass `qw.msg.Host()` instead of `qw.msg.RemoteAddr()`

### Update callers of processor message constructors

- [x] update: [pkg/vvm/impl_requesthandler.go](../../../pkg/vvm/impl_requesthandler.go)
  - update: Remove `request.RemoteAddr` from `NewCommandMessage`, `NewQueryMessage`, and `NewIQueryMessage` calls (already use `request.Host`)

### Update `ResetRateLimit` callers

- [x] update: [pkg/sys/verifier/impl.go](../../../pkg/sys/verifier/impl.go)
  - update: Anonymous `ResetRateLimit` interface cast and call to include `remoteAddr string` parameter

### Update mocks and tests

- [x] update: [pkg/processors/schedulers/impl_test.go](../../../pkg/processors/schedulers/impl_test.go)
  - update: Mock `ResetRateLimit` signature to include `remoteAddr string`
- [x] update: [pkg/processors/command/impl_test.go](../../../pkg/processors/command/impl_test.go)
  - update: `NewCommandMessage` calls to include `remoteAddr` parameter
- [x] Review


### Integration test for PER IP rate limit

- [x] update: [pkg/vit/schemaTestApp1.vsql](../../../pkg/vit/schemaTestApp1.vsql)
  - add: `IPRatedCmd` command and `IPRatedQry` query declarations
  - add: `RATE IPRatedPerMinute 2 PER MINUTE PER IP` and corresponding `LIMIT` declarations
- [x] update: [pkg/vit/shared_cfgs.go](../../../pkg/vit/shared_cfgs.go)
  - add: `QNameCmdIPRated` and `QNameQryIPRated` QName vars
  - add: Builtin function registrations for `IPRatedCmd` and `IPRatedQry`
- [x] update: [pkg/sys/it/impl_rates_test.go](../../../pkg/sys/it/impl_rates_test.go)
  - add: `TestRates_PerIP` — verifies PER IP rate limits using `127.0.0.1` (localhost test requests)
- [x] Review