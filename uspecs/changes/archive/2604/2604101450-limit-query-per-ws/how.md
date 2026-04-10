# How: Limit concurrent query executions per workspace

## Approach

- Add a `wsQueryLimiter` type in `pkg/router` with a `sync.Map` keyed by `istructs.WSID`, values are `*atomic.Int32` counters, and `maxQPerWS int` field encapsulating the limit
- Expose two methods: `acquire(wsid) bool` (increment-if-below-limit, return false if at limit; if `maxQPerWS <= 0`, treat it as unlimited and return true) and `release(wsid)` (decrement)
- Add `MaxQueriesPerWS int` field to `RouterParams` in `pkg/router/types.go`, defaulting to 10 via `DefaultMaxQueriesPerWSLimit` constant in `pkg/router/consts.go`
  - use it in http\https service only, not in admin or ACME services
- Store the limiter instance as a `*wsQueryLimiter` pointer field on `routerService` in `pkg/router/types.go`
- Wire the limit value through `provideRouterParams` in `pkg/vvm/provide.go` from `VVMConfig`
- Add parameter `RouterMaxQueriesPerWS` to `VVMConfig` in `pkg/vvm/types.go`, following the existing `RouterWriteTimeout`/`RouterReadTimeout`/`RouterConnectionsLimit` pattern

- **V2 detection**: use `busRequest.APIPath` to identify QP-bound requests. The limiter applies to handlers where `APIPath` is `APIPath_Queries`, `APIPath_Views`, `APIPath_Docs`, or `APIPath_CDocs` — these are the paths routed to the query processor by `impl_requesthandler.go`. Schema reads are excluded (no WSID). Blobs and notifications use separate handlers
- **V2 placement**: inject the limiter call inside `sendRequestAndReadResponse` in `pkg/router/impl_apiv2.go`, checking `busRequest.APIPath` via `isQPBoundAPIPath()` helper. This centralizes the logic in one place rather than modifying every handler
- **V1 detection**: in `RequestHandler_V1` in `pkg/router/impl_http.go`, check `limiter != nil && busRequest.Resource[:1] == "q"` before calling `SendRequest`. `nil` limiter check supports `ihttpimpl` caller which passes `nil` (no limiting for internal HTTP processor)
- On limit exceeded, return HTTP 503 with `Retry-After: <DefaultRetryAfterSecondsOn503>` header via `replyServiceUnavailable()` (fixed: header must be set before `WriteHeader`)
- Decrement via `defer limiter.release(wsid)` immediately after successful `acquire`, before `SendRequest` call — `reply_v1`/`reply_v2` blocking guarantees the QP slot is held until response is fully consumed

References:

- [pkg/router/impl_apiv2.go](../../../../pkg/router/impl_apiv2.go)
- [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)
- [pkg/router/types.go](../../../../pkg/router/types.go)
- [pkg/router/consts.go](../../../../pkg/router/consts.go)
- [pkg/vvm/provide.go](../../../../pkg/vvm/provide.go)
- [pkg/vvm/types.go](../../../../pkg/vvm/types.go)
- [pkg/vvm/impl_requesthandler.go](../../../../pkg/vvm/impl_requesthandler.go)
- [pkg/processors/consts.go](../../../../pkg/processors/consts.go)
