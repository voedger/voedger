# How: Send Retry-After header on function limit exceed

## Approach

- Compute Retry-After from VSQL-declared rate parameters, not from bucket internals:
  - Reuse the existing `(exceeded, limitQName)` return of `appPart.IsLimitExceeded` in `pkg/appparts/interface.go`
  - Look up `appdef.Limit(appDef.Type, limitQName)` and read `Rate().Count()` and `Rate().Period()` — the same access pattern as `vit.RatePerPeriod()` in `pkg/vit/utils.go`
  - Use `Period / Count` as the per-token interval; apply `math.Ceil` to seconds and enforce a 1-second minimum
  - `IAppDef` is already reachable in each processor via `appPart.AppStructs().AppDef()`
- Carry the computed duration from processor to router via `coreutils.SysError`:
  - Extend `coreutils.SysError` in `pkg/coreutils/syserror.go` with an optional `headers map[string]string` field and a method `AddHeader(key, value string) SysError` that returns the enriched `SysError`
  - Rate-limit check functions build a 429 `SysError` and call `.AddHeader("Retry-After", strconv.Itoa(ceilSeconds))` before returning it; the existing `bus.ReplyErr` → `responder.Respond` → router path already carries the `SysError` through to the router
  - In `pkg/router/impl_http.go` `initResponse` (and any other writer that materializes a `SysError` response), after matching/unwrapping the `SysError`, iterate its `Headers` and `w.Header().Set(k, v)` before `WriteHeader`
- Update the three 429-producing processors to compute and propagate the value:
  - `pkg/processors/command/impl.go` `limitCallRate`
  - `pkg/processors/query/impl.go` "check function call rate" inline operator
  - `pkg/processors/query2/util.go` `queryRateLimitExceeded`
  - Extract a small shared helper (e.g. in `pkg/processors/impl_ratelimit.go` or similar existing `pkg/processors` shared package) to avoid duplication: given `appPart`, exceeded `limitQName`, return the `time.Duration` for Retry-After
- Keep `replyServiceUnavailable` in `pkg/router/utils.go` and the 503 path in `pkg/vvm/impl_requesthandler.go` unchanged — they are out of scope and already send `Retry-After`
- Tests:
  - Extend `TestRateLimit` in `pkg/processors/command/impl_test.go` to assert that the 429 response carries a `Retry-After` header with a value consistent with `rateName`'s `Count/Period`
  - Add analogous assertions in query v1 and v2 rate-limit tests
  - Add a unit test for the Retry-After computation helper covering: integer seconds, sub-second period rounded up, 1-second floor

References:

- [pkg/appparts/interface.go](../../../../../pkg/appparts/interface.go)
- [pkg/appparts/internal/limiter/limiter.go](../../../../../pkg/appparts/internal/limiter/limiter.go)
- [pkg/appdef/interface_ratelimit.go](../../../../../pkg/appdef/interface_ratelimit.go)
- [pkg/vit/utils.go](../../../../../pkg/vit/utils.go)
- [pkg/coreutils/syserror.go](../../../../../pkg/coreutils/syserror.go)
- [pkg/bus/utils.go](../../../../../pkg/bus/utils.go)
- [pkg/router/impl_http.go](../../../../../pkg/router/impl_http.go)
- [pkg/router/impl_reply_v1.go](../../../../../pkg/router/impl_reply_v1.go)
- [pkg/router/impl_reply_v2.go](../../../../../pkg/router/impl_reply_v2.go)
- [pkg/router/utils.go](../../../../../pkg/router/utils.go)
- [pkg/processors/command/impl.go](../../../../../pkg/processors/command/impl.go)
- [pkg/processors/query/impl.go](../../../../../pkg/processors/query/impl.go)
- [pkg/processors/query2/util.go](../../../../../pkg/processors/query2/util.go)
- [pkg/processors/command/impl_test.go](../../../../../pkg/processors/command/impl_test.go)
