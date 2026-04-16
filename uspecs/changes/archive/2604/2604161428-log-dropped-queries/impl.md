# Implementation plan: Log dropped queries per workspace limit

## Technical design

- [x] update: [logging--td.md](../../../../specs/prod/apps/logging--td.md)
  - update: replace `routing.qpLimiterSize` entry with `routing.qp.limit` warning entry: level `Warning`, stage `routing.qp.limit`, msg `droppedInLast10Seconds=<count>`
  - add: description of deferred aggregation mechanism (per [wsid, extension] key, 10s flush interval, flush on shutdown)

## Construction

- [x] update: [pkg/router/types.go](../../../../../../../../pkg/router/types.go)
  - add: `rejectionKey{wsid istructs.WSID, extension string}` type
  - add: `rejectionCounter{count int64, logCtxFromLastQuery context.Context}` type
  - update: `wsQueryLimiter` — add `mu sync.Mutex`, `rejections map[rejectionKey]*rejectionCounter`, `lastLoggedAt int64`, `iTime timeu.ITime` fields
  - update: `RouterParams` — add `ITime timeu.ITime` field

- [x] update: [pkg/router/impl_limiter.go](../../../../../../../../pkg/router/impl_limiter.go)
  - add: `onQueryDrop(requestCtx, wsid, extension)` — under mutex: bump counter per key, store `requestCtx`, set `lastLoggedAt` on first drop
  - add: `tryFlush()` — under mutex: check `lastLoggedAt`, swap map, update timestamp; log outside lock
  - add: `flushAll()` — under mutex: swap map, reset `lastLoggedAt`; log outside lock
  - add: `logRejections(entries)` — log one warning per entry with non-zero count
  - remove: `size()` method (no longer needed)

- [x] update: [pkg/router/utils.go](../../../../../../../../pkg/router/utils.go)
  - add: `resolveExtension(busRequest)` helper extracted from `withLogAttribs()`
  - update: `withLogAttribs()` — use `resolveExtension()`
  - update: `logServeRequest()` — remove `routing.qpLimiterSize` log; add `limiter.tryFlush()` call

- [x] update: [pkg/router/consts.go](../../../../../../../../pkg/router/consts.go)
  - add: `rejectionLogInterval = 10 * time.Second`
  - remove: `limiterSizeLogIntervalInRequests`

- [x] update: [pkg/router/provide.go](../../../../../../../../pkg/router/provide.go)
  - update: `getRouterService()` — pass `rp.ITime` to `wsQueryLimiter`, initialize `rejections` map

- [x] update: [pkg/router/impl_apiv2.go](../../../../../../../../pkg/router/impl_apiv2.go)
  - update: `sendRequestAndReadResponse()` — move `withLogAttribs` before limiter check; call `limiter.onQueryDrop()` on rejection

- [x] update: [pkg/router/impl_http.go](../../../../../../../../pkg/router/impl_http.go)
  - update: `RequestHandler_V1()` — move `withLogAttribs` before limiter check; call `limiter.onQueryDrop()` on rejection
  - update: `routerService.Stop()` — call `queryLimiter.flushAll()` before `httpServer.Stop()`

- [x] update: [pkg/router/impl_blob.go](../../../../../../../../pkg/router/impl_blob.go)
  - no changes needed — `logServeRequest` still takes `limiter` param for `tryFlush()` calls

- [x] update: [pkg/vvm/provide.go](../../../../../../../../pkg/vvm/provide.go)
  - update: `provideRouterParams()` — pass `cfg.Time` as `ITime` field

- [x] update: [pkg/sys/it/impl_rates_test.go](../../../../../../../../pkg/sys/it/impl_rates_test.go)
  - update: existing `qpv1`/`qpv2` subtests — add `logger.StartCapture`, `vit.TimeAdd(10*time.Second)`, trigger flush query, assert `HasLine("stage=routing.qp.limit", "droppedInLast10Seconds=1")`

- [x] Review
