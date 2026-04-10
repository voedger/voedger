# Implementation plan: Limit concurrent query executions per workspace

## Construction

### Core limiter

- [x] create: [pkg/router/impl_limiter.go](../../../../pkg/router/impl_limiter.go)
  - add: `wsQueryLimiter` type with `sync.Map` keyed by `istructs.WSID`, `*atomic.Int32` values, and `maxQPerWS int` field
  - add: `acquire(wsid) bool` method — atomically increment if below `maxQPerWS`, return true if `maxQPerWS <= 0`
  - add: `release(wsid)` method — atomically decrement counter
  - add: `isQPBoundAPIPath(apiPath) bool` helper for V2 detection

### Configuration

- [x] update: [pkg/router/consts.go](../../../../pkg/router/consts.go)
  - add: `DefaultMaxQueriesPerWSLimit` constant with value 10
- [x] update: [pkg/router/types.go](../../../../pkg/router/types.go)
  - add: `MaxQueriesPerWS int` field to `RouterParams`
  - add: `queryLimiter *wsQueryLimiter` field to `routerService`
- [x] update: [pkg/vvm/types.go](../../../../pkg/vvm/types.go)
  - add: `RouterMaxQueriesPerWS int` field to `VVMConfig`
- [x] update: [pkg/vvm/impl_cfg.go](../../../../pkg/vvm/impl_cfg.go)
  - add: default value `RouterMaxQueriesPerWS: router.DefaultMaxQueriesPerWSLimit` in `NewVVMDefaultConfig()`
- [x] update: [pkg/vvm/provide.go](../../../../pkg/vvm/provide.go)
  - add: `MaxQueriesPerWS: cfg.RouterMaxQueriesPerWS` in `provideRouterParams()`
- [x] update: [pkg/vvm/wire_gen.go](../../../../pkg/vvm/wire_gen.go)
  - add: `MaxQueriesPerWS: cfg.RouterMaxQueriesPerWS` in `provideRouterParams()`
- [x] update: [pkg/router/provide.go](../../../../pkg/router/provide.go)
  - add: `queryLimiter: &wsQueryLimiter{maxQPerWS: rp.MaxQueriesPerWS}` in `getRouterService()`

### V2 limiter integration

- [x] update: [pkg/router/impl_apiv2.go](../../../../pkg/router/impl_apiv2.go)
  - update: `sendRequestAndReadResponse()` — accept `limiter *wsQueryLimiter`; before `SendRequest`, check `isQPBoundAPIPath(busRequest.APIPath)` and if so, call `limiter.acquire(busRequest.WSID)`; on failure call `replyServiceUnavailable(rw)` and return; on success defer `limiter.release(busRequest.WSID)`
  - update: all `requestHandlerV2_*` factory functions — accept `limiter *wsQueryLimiter` and pass to `sendRequestAndReadResponse`

### V1 limiter integration

- [x] update: [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)
  - update: `RequestHandler_V1()` — accept `limiter *wsQueryLimiter`; before `SendRequest`, check `limiter != nil && busRequest.Resource[:1] == "q"` and if so, call `limiter.acquire(busRequest.WSID)`; on failure call `replyServiceUnavailable(rw)` and return; on success defer `limiter.release(busRequest.WSID)`
- [x] update: [pkg/ihttpimpl/impl.go](../../../../pkg/ihttpimpl/impl.go)
  - update: pass `nil` for limiter in `RequestHandler_V1` call (no limiting for internal HTTP processor)

### Bug fix

- [x] update: [pkg/router/utils.go](../../../../pkg/router/utils.go)
  - fix: `replyServiceUnavailable()` — set `Retry-After` header before `WriteHeader` (Go ignores headers set after `WriteHeader`)

### Tests

- [x] update: [pkg/sys/it/impl_rates_test.go](../../../../pkg/sys/it/impl_rates_test.go)
  - add: `TestQueryLimiter_BasicUsage` integration test using shared VIT config with `vit.VVMConfig.RouterMaxQueriesPerWS`
  - add: subtest: queries rejected with 503 when per-workspace limit reached (V1 and V2)
  - add: subtest: commands not affected by the limiter
  - add: subtest: different workspaces independently limited
  - add: helpers `fillQuerySlots`/`releaseQuerySlots` using `MockQryExec`
