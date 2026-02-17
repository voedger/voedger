# Implementation plan

## Construction

### Bus package (foundational changes)

- [x] update: [pkg/bus/consts.go](../../../pkg/bus/consts.go)
  - remove: `DefaultSendTimeout` exported constant
  - add: `const sendResponseTimeout = 10 * time.Second` — how long `Write()` waits before returning `ErrSendResponseTimeout`
  - add: `const firstResponseWaitWarningInterval = time.Minute`

- [x] update: [pkg/bus/errors.go](../../../pkg/bus/errors.go)
  - remove: `ErrSendTimeoutExpired` error variable
  - rename: `ErrNoConsumer` → `ErrSendResponseTimeout` with message "timeout sending response"

- [x] update: [pkg/bus/types.go](../../../pkg/bus/types.go)
  - remove: `SendTimeout` type (no longer configurable)
  - remove: `timeout` field from `implIRequestSender` (no longer used by `SendRequest()`)
  - remove: `sendTimeout` field from `implResponseWriter` (replaced by `sendResponseTimeout` const)

- [x] update: [pkg/bus/interface.go](../../../pkg/bus/interface.go)
  - remove: `ErrSendTimeoutExpired` from `SendRequest` doc comment

- [x] update: [pkg/bus/provide.go](../../../pkg/bus/provide.go)
  - remove: `sendTimeout` parameter from `NewIRequestSender`

- [x] update: [pkg/bus/impl.go](../../../pkg/bus/impl.go)
  - update: `SendRequest()` — remove `timeoutChan`; merge response-awaiting and warning into single goroutine via `wg.Go()`; use `time.NewTicker(firstResponseWaitWarningInterval)` for periodic warnings with elapsed time logging; stop passing `sendTimeout` to `implResponseWriter`
  - update: `Write()` — use `sendResponseTimeout` const instead of `rs.sendTimeout` field; return `ErrSendResponseTimeout`
  - update: `Respond()` — replace `default: return ErrNoConsumer` with `<-r.respWriter.clientCtx.Done(): return r.respWriter.clientCtx.Err()`
  - update: `StreamJSON()` and `StreamEvents()` — replace `select/default` with direct channel send

- [x] update: [pkg/bus/impl_test.go](../../../pkg/bus/impl_test.go)
  - remove: "response timeout" test (`SendRequest` timeout no longer exists)
  - rename: "no consumer" test → "send response timeout"; `ErrNoConsumer` → `ErrSendResponseTimeout`
  - update: all `NewIRequestSender` calls to remove `sendTimeout` argument

- [x] update: [pkg/bus/utils_test.go](../../../pkg/bus/utils_test.go)
  - remove: `sendTimeout` constant
  - update: all `NewIRequestSender` calls to remove `sendTimeout` argument

- [x] update: [pkg/bus/README.md](../../../pkg/bus/README.md)
  - remove: "timeout" alternative from sequence diagram (`SendRequest` no longer times out)

### Router package (dependent on bus)

- [x] update: [pkg/router/impl_http.go](../../../pkg/router/impl_http.go)
  - remove: `ErrSendTimeoutExpired` check and `StatusServiceUnavailable` mapping in `RequestHandler_V1`

- [x] update: [pkg/router/impl_apiv2.go](../../../pkg/router/impl_apiv2.go)
  - remove: `ErrSendTimeoutExpired` check and `StatusServiceUnavailable` mapping in `sendRequestAndReadResponse`

- [x] update: [pkg/router/consts.go](../../../pkg/router/consts.go)
  - update: `DefaultRouterWriteTimeout` to 0 (disable HTTP server write timeout)
  - add: comments explaining `DefaultRouterReadTimeout` and `DefaultRouterWriteTimeout`

- [x] update: [pkg/router/impl_test.go](../../../pkg/router/impl_test.go)
  - remove: `TestBeginResponseTimeout` test (`SendRequest` timeout no longer exists)
  - remove: `sendTimeout` field from `testRouter` struct
  - remove: `sendTimeout bus.SendTimeout` parameter from `setUp` and `startRouter` functions
  - update: all `setUp` and `startRouter` callers to remove `sendTimeout` argument

### VVM configuration (dependent on bus)

- [x] update: [pkg/vvm/types.go](../../../pkg/vvm/types.go)
  - remove: `SendTimeout` field from `VVMConfig`

- [x] update: [pkg/vvm/impl_cfg.go](../../../pkg/vvm/impl_cfg.go)
  - remove: `SendTimeout` assignment from `NewVVMDefaultConfig`

- [x] update: [pkg/vvm/provide.go](../../../pkg/vvm/provide.go)
  - remove: `"SendTimeout"` from `wire.FieldsOf`
  - remove: `sendTimeout bus.SendTimeout` parameter from `provideRouterServices`

- [x] update: [pkg/vvm/wire_gen.go](../../../pkg/vvm/wire_gen.go)
  - remove: `sendTimeout := vvmConfig.SendTimeout` line
  - remove: `sendTimeout` parameter from `bus.NewIRequestSender` call
  - remove: `sendTimeout` parameter from `provideRouterServices` call and signature

### VIT test infrastructure (dependent on VVM)

- [x] update: [pkg/vit/impl.go](../../../pkg/vit/impl.go)
  - remove: `cfg.SendTimeout` assignment

### Downstream callers of `NewIRequestSender` (remove `sendTimeout` argument)

- [x] update: [pkg/ihttpimpl/provide.go](../../../pkg/ihttpimpl/provide.go)
  - remove: `bus.DefaultSendTimeout` argument from `bus.NewIRequestSender` call

- [x] update: [pkg/processors/command/impl_test.go](../../../pkg/processors/command/impl_test.go)
  - remove: `sendTimeout` argument from `bus.NewIRequestSender` call

- [x] update: [pkg/processors/query/impl_test.go](../../../pkg/processors/query/impl_test.go)
  - remove: `sendTimeout` constant
  - remove: `sendTimeout` argument from all `bus.NewIRequestSender` calls

- [x] update: [pkg/processors/query/operator-send-to-bus-impl_test.go](../../../pkg/processors/query/operator-send-to-bus-impl_test.go)
  - remove: `sendTimeout` argument from `bus.NewIRequestSender` call

- [x] update: [pkg/processors/query/query-params-impl_test.go](../../../pkg/processors/query/query-params-impl_test.go)
  - remove: `sendTimeout` argument from `bus.NewIRequestSender` call

- [x] update: [pkg/sys/collection/collection_test.go](../../../pkg/sys/collection/collection_test.go)
  - remove: `sendTimeout` constant
  - remove: `sendTimeout` argument from all `bus.NewIRequestSender` calls
