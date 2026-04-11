# Implementation plan: Suppress processor submit failure logging in tests

## Construction

- [x] update: [pkg/vvm/consts.go](../../../../../../../pkg/vvm/consts.go)
  - add: `BusyProcessorLogMode` type (`int`) with constants `BusyProcessorLogMode_Error` (default) and `BusyProcessorLogMode_Silent`
- [x] update: [pkg/vvm/types.go](../../../../../../../pkg/vvm/types.go)
  - add: `BusyProcessorLogMode BusyProcessorLogMode` field to `VVMConfig`
- [x] update: [pkg/vvm/impl_cfg.go](../../../../../../../pkg/vvm/impl_cfg.go)
  - add: Set `BusyProcessorLogMode: BusyProcessorLogMode_Error` in `NewVVMDefaultConfig()`
- [x] update: [pkg/vvm/provide.go](../../../../../../../pkg/vvm/provide.go)
  - add: `"BusyProcessorLogMode"` to `wire.FieldsOf`
  - update: `provideRequestHandler` signature to accept `BusyProcessorLogMode` parameter
  - update: Pass `BusyProcessorLogMode` to `replyCommandBusy` and `replyQueryBusy`
- [x] update: [pkg/vvm/impl_requesthandler.go](../../../../../../../pkg/vvm/impl_requesthandler.go)
  - update: `replyCommandBusy` and `replyQueryBusy` to accept `BusyProcessorLogMode` and skip `logger.ErrorCtx` when `BusyProcessorLogMode_Silent`
- [x] update: [pkg/vvm/wire_gen.go](../../../../../../../pkg/vvm/wire_gen.go)
  - update: Pass `BusyProcessorLogMode` from `VVMConfig` to `provideRequestHandler`
- [x] update: [pkg/vit/impl.go](../../../../../../../pkg/vit/impl.go)
  - update: Set `cfg.BusyProcessorLogMode = vvm.BusyProcessorLogMode_Silent` in `newVit()`
- [x] update: [pkg/sys/it/impl_test.go](../../../../../../../pkg/sys/it/impl_test.go)
  - update: Remove `logger.StartCapture` and `logCap.HasLine` from `Test503OnNoCommandProcessorsAvailable` and `Test503OnNoQueryProcessorsAvailable` (logging is now suppressed in tests)
