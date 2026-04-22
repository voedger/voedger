# How: Use PER IP rate limit scope

## Approach

- Two fixes needed: `ResetLimits` missing `RateScope_IP` in bucket key construction, and all `IsLimitExceeded` callers passing empty `remoteAddr`
- For `ResetLimits` in `pkg/appparts/internal/limiter/limiter.go` — add `remoteAddr string` parameter and set `key.RemoteAddr` when `RateScope_IP`, mirroring the existing logic in `Exceeded`
- Propagate `remoteAddr` through the interface chain: `IAppPartition.ResetRateLimit` in `pkg/appparts/interface.go` gains `remoteAddr` parameter, implementation in `pkg/appparts/impl_app.go` forwards it
- Source of `remoteAddr`: repurpose the existing `Host` field on `bus.Request` in `pkg/bus/types.go` to carry the client IP address (host only, port stripped). The router sets `Host: remoteIP(req.RemoteAddr)` in `pkg/router/utils.go` (`createBusRequest`), stripping the port via `net.SplitHostPort`. No separate `RemoteAddr` field is needed since `Host` was previously unused by the router
- The `Host` field is already threaded through the VVM request handler into processor message constructors (`NewCommandMessage`, `NewQueryMessage`, `NewIQueryMessage`) and exposed via `Host()` accessors
- Processor rate limit checks pass `msg.Host()` instead of `""`: `limitCallRate` in command processor, inline check in query v1 processor, `queryRateLimitExceeded` in query2
- The verifier in `pkg/sys/verifier/impl.go` uses an anonymous interface cast for `ResetRateLimit` — update the interface signature to include `remoteAddr`
- Update mocks in `pkg/processors/schedulers/impl_test.go` and test call sites in `pkg/processors/command/impl_test.go`

**Investigation needed:**

- `http.Request.RemoteAddr` contains `IP:port` — the port is stripped via `net.SplitHostPort` in the router before propagating to `bus.Request.RemoteAddr`. The value may still reflect a reverse proxy address rather than the real client IP
- Consider whether headers like `X-Forwarded-For` or `X-Real-IP` should be preferred when a reverse proxy is in front of the router
- Ensure the router extracts the correct value and propagates it via `bus.Request` to the VVM before processors use it for IP-scoped rate limiting

References:

- [pkg/appparts/internal/limiter/limiter.go](../../../../../pkg/appparts/internal/limiter/limiter.go)
- [pkg/appparts/interface.go](../../../../../pkg/appparts/interface.go)
- [pkg/appparts/impl_app.go](../../../../../pkg/appparts/impl_app.go)
- [pkg/bus/types.go](../../../../../pkg/bus/types.go)
- [pkg/router/utils.go](../../../../../pkg/router/utils.go)
- [pkg/vvm/impl_requesthandler.go](../../../../../pkg/vvm/impl_requesthandler.go)
- [pkg/processors/command/types.go](../../../../../pkg/processors/command/types.go)
- [pkg/processors/command/impl.go](../../../../../pkg/processors/command/impl.go)
- [pkg/processors/query/impl.go](../../../../../pkg/processors/query/impl.go)
- [pkg/processors/query2/types.go](../../../../../pkg/processors/query2/types.go)
- [pkg/processors/query2/util.go](../../../../../pkg/processors/query2/util.go)
- [pkg/sys/verifier/impl.go](../../../../../pkg/sys/verifier/impl.go)

