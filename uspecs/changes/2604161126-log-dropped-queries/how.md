# How: Log dropped queries per workspace limit

## Approach

- Add `rejectionKey{wsid, extension}` and `rejectionCounter{count int64, logCtxFromLastQuery context.Context}` types to `pkg/router/types.go`
- Add `mu sync.Mutex`, `rejections map[rejectionKey]*rejectionCounter`, `lastLoggedAt int64`, and `iTime timeu.ITime` fields to `wsQueryLimiter` in `pkg/router/types.go`
- Extract `resolveExtension(busRequest)` helper from `withLogAttribs()` in `pkg/router/utils.go` to avoid duplicating extension resolution logic
- Move `withLogAttribs` call before the limiter check in both `sendRequestAndReadResponse()` and `RequestHandler_V1()` so the enriched context is available at rejection time
- On rejection: call `limiter.onQueryDrop(requestCtx, wsid, extension)` under mutex — increment `count`, store `requestCtx`, set `lastLoggedAt` to `iTime.Now().UnixNano()` on first drop
- On every query: call `limiter.tryFlush()` from `logServeRequest()` — acquire mutex, check if `lastLoggedAt` is non-zero and 10s have elapsed per `iTime.Now()`, swap the rejections map, update `lastLoggedAt`, release mutex, log outside the lock. Use `ITime` to enable mocked time in tests
- Add `flushAll()` — swap rejections map under mutex, reset `lastLoggedAt` to 0, log outside the lock
- Extract `logRejections(entries)` helper to deduplicate logging logic between `tryFlush` and `flushAll`
- Call `limiter.flushAll()` in `routerService.Stop()` before `httpServer.Stop()`
- Remove `routing.qpLimiterSize` verbose log from `logServeRequest` and `limiterSizeLogIntervalInRequests` constant
- Update existing test `TestQueryLimiter_BasicUsage` — add log capture, advance mock time by 10s, trigger flush query, assert `HasLine("stage=routing.qp.limit", "droppedInLast10Seconds=1")`

References:

- [pkg/router/types.go](../../../../pkg/router/types.go)
- [pkg/router/impl_limiter.go](../../../../pkg/router/impl_limiter.go)
- [pkg/router/impl_apiv2.go](../../../../pkg/router/impl_apiv2.go)
- [pkg/router/impl_http.go](../../../../pkg/router/impl_http.go)
- [pkg/router/utils.go](../../../../pkg/router/utils.go)
- [pkg/router/consts.go](../../../../pkg/router/consts.go)
- [pkg/sys/it/impl_rates_test.go](../../../../pkg/sys/it/impl_rates_test.go)
