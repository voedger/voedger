# Implementation plan

## Construction

### HTTP client lifecycle ownership

- [x] update: [pkg/vvm/provide.go](../../../pkg/vvm/provide.go)
  - update: `provideStateOpts` to create `httpu.IHTTPClient` via `httpu.NewIHTTPClient()`, return `(state.StateOpts, func())` with client set in `StateOpts.CustomHTTPClient` and cleanup as second return value

### StateOpts threading

- [x] update: [pkg/processors/schedulers/interface.go](../../../pkg/processors/schedulers/interface.go)
  - update: export `stateOpts` field to `StateOpts` in `BasicSchedulerConfig`
- [x] update: [pkg/processors/schedulers/impl_scheduler.go](../../../pkg/processors/schedulers/impl_scheduler.go)
  - update: `a.conf.stateOpts` → `a.conf.StateOpts`
- [x] update: [pkg/processors/query/provide.go](../../../pkg/processors/query/provide.go)
  - update: `ProvideServiceFactory` to accept `state.StateOpts` and return a closure capturing it
- [x] update: [pkg/processors/query/impl.go](../../../pkg/processors/query/impl.go)
  - update: `implServiceFactory` and `newQueryProcessorPipeline` to accept and use `state.StateOpts` instead of `state.NullOpts`
- [x] update: [pkg/processors/query2/provide.go](../../../pkg/processors/query2/provide.go)
  - update: `ProvideServiceFactory` to accept `state.StateOpts` and return a closure capturing it
- [x] update: [pkg/processors/query2/impl.go](../../../pkg/processors/query2/impl.go)
  - update: `implServiceFactory` and `newQueryProcessorPipeline` to accept and use `state.StateOpts` instead of `state.NullOpts`
- [x] update: [pkg/vvm/wire_gen.go](../../../pkg/vvm/wire_gen.go)
  - update: `provideStateOpts` call to capture cleanup and add it to the cleanup chain
  - update: `BasicSchedulerConfig` to set `StateOpts` field
  - update: `queryprocessor.ProvideServiceFactory` and `query2.ProvideServiceFactory` calls to pass `stateOpts`

### Storage simplification

- [x] update: [pkg/sys/storages/impl_http_storage.go](../../../pkg/sys/storages/impl_http_storage.go)
  - remove: `cleanup` field from `httpStorage` struct
  - update: `NewHTTPStorage` to require non-nil `customClient` parameter (remove nil-handling and internal client creation)

### Test updates

- [x] update: [pkg/sys/storages/impl_http_storage_test.go](../../../pkg/sys/storages/impl_http_storage_test.go)
  - update: tests that pass `nil` to `NewHTTPStorage` to create `httpu.IHTTPClient` explicitly and pass it
- [ ] Review
