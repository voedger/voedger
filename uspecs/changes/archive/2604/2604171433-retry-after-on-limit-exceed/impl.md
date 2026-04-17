# Implementation plan: Send Retry-After header on function limit exceed

## Construction

### SysError headers support

- [x] update: [pkg/coreutils/syserror.go](../../../../../pkg/coreutils/syserror.go)
  - add: unexported `headers map[string]string` field on `SysError`
  - add: `AddHeader(key, value string) SysError` method returning the enriched value. Make it panic if the key is set already
  - add: `Headers() map[string]string` accessor
  - ensure: `ToJSON_APIV1`, `ToJSON_APIV2` and `IsNil` remain unaffected by the new field
- [x] update: [pkg/coreutils/syserror_test.go](../../../../../pkg/coreutils/syserror_test.go)
  - add: test for `AddHeader` / `Headers` / error wrapping preserves headers
- [x] Review

### Retry-After computation helper

- [x] update: [pkg/processors/utils.go](../../../../../pkg/processors/utils.go)
  - add: helper `RetryAfterSecondsOnLimitExceeded(appDef appdef.IAppDef, limit appdef.QName) int` — reads `Rate().Count()`/`Rate().Period()` via `appdef.Limit(appDef.Type, limit)` and returns `ceil(Period.Seconds()/Count)` with a minimum of 1
- [x] create: [pkg/processors/utils_test.go](../../../../../pkg/processors/utils_test.go)
  - add: subtests for integer seconds, sub-second period rounded up, 1-second floor
- [x] Review

### Router: apply SysError headers on response

- [x] update: [pkg/router/impl_http.go](../../../../../pkg/router/impl_http.go)
  - update: `initResponse` no longer calls `WriteHeader`; added helper `applySysErrorHeaders`
- [x] update: [pkg/router/impl_reply_v1.go](../../../../../pkg/router/impl_reply_v1.go)
  - update: signature now takes `bus.ResponseMeta`; `WriteHeader(statusCode)` is deferred via a `writeHeaderOnce` closure in both Single and multi (StreamJSON) modes. On `coreutils.SysError` the headers are applied via `applySysErrorHeaders` before `WriteHeader` — including when the error arrives via `responseErr` after zero elements
- [x] update: [pkg/router/impl_reply_v2.go](../../../../../pkg/router/impl_reply_v2.go)
  - update: `WriteHeader(statusCode)` is deferred via a `writeHeaderOnce` closure. On `coreutils.SysError` the headers are applied via `applySysErrorHeaders` before `WriteHeader`
- [x] update: [pkg/router/impl_test.go](../../../../../pkg/router/impl_test.go)
  - add: `TestSysErrorHeaders` asserts that a `SysError` with `Retry-After` header results in `Retry-After` on the HTTP response
- [x] Review

### Processors: attach Retry-After on 429

- [x] update: [pkg/processors/command/impl.go](../../../../../pkg/processors/command/impl.go)
  - update: `limitCallRate` — on exceeded, builds `coreutils.NewHTTPErrorf(http.StatusTooManyRequests).AddHeader(httpu.RetryAfter, strconv.Itoa(processors.RetryAfterSecondsOnLimitExceeded(cmd.appStructs.AppDef(), limit)))`
- [x] update: [pkg/processors/command/impl_test.go](../../../../../pkg/processors/command/impl_test.go)
  - update: `TestRateLimit` — asserts `Retry-After` on the 429 `SysError` via `requestSender.SendRequest` (value `30` for 2/min rate)
- [x] update: [pkg/processors/query/impl.go](../../../../../pkg/processors/query/impl.go)
  - update: inline "check function call rate" operator — attach `Retry-After` to the 429 `SysError` the same way
- [x] update: [pkg/processors/query/impl_test.go](../../../../../pkg/processors/query/impl_test.go)
  - update: `TestRateLimiter` — asserts `Retry-After` on the 429 `*respErr` via `errors.As` (value `30` for 2/min rate)
- [x] update: [pkg/processors/query2/util.go](../../../../../pkg/processors/query2/util.go)
  - update: `queryRateLimitExceeded` — attaches `Retry-After` to the 429 `SysError` the same way
- [-] update: [pkg/processors/query2/impl_test.go](../../../../../pkg/processors/query2/impl_test.go)
  - rationale: query2 has no existing unit-level test for rate limits; the integration test in `pkg/sys/it/impl_rates_test.go` covers the end-to-end API v2 path using the same helper
- [x] Review

### Integration test

- [x] update: [pkg/sys/it/impl_rates_test.go](../../../../../pkg/sys/it/impl_rates_test.go)
  - update: `TestRates_BasicUsage` asserts `Retry-After: 30` on the first per-minute 429 (both query and command) and `Retry-After: 900` on the per-hour 429
  - update: `TestRates_PerIP` asserts `Retry-After: 30` on the per-IP 429
- [x] Review
