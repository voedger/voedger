# Implementation plan: Make command processor channel buffer size configurable

## Construction

- [x] update: [pkg/vvm/consts.go](../../../../../pkg/vvm/consts.go)
  - add: `DefaultCommandProcessorChannelBufferSize uint = 0` constant
- [x] update: [pkg/vvm/types.go](../../../../../pkg/vvm/types.go)
  - add: `CommandProcessorChannelBufferSize uint` field to `VVMConfig` struct
- [x] update: [pkg/vvm/impl_cfg.go](../../../../../pkg/vvm/impl_cfg.go)
  - add: Initialize `CommandProcessorChannelBufferSize` with `DefaultCommandProcessorChannelBufferSize` in `NewVVMDefaultConfig()`
- [x] update: [pkg/vvm/provide.go](../../../../../pkg/vvm/provide.go)
  - update: Use `vvmCfg.CommandProcessorChannelBufferSize` instead of `uint(DefaultNumCommandProcessors)` for command processor `ChannelBufferSize`
- [x] update: [pkg/vvm/wire_gen.go](../../../../../pkg/vvm/wire_gen.go)
  - update: Use `vvmCfg.CommandProcessorChannelBufferSize` instead of `uint(DefaultNumCommandProcessors)` for command processor `ChannelBufferSize`
- [x] update: [pkg/sys/it/impl_test.go](../../../../../pkg/sys/it/impl_test.go)
  - add: `Test503OnNoCommandProcessorsAvailable` — integration test verifying that with buffer size 0, a 2nd concurrent command to the same partition returns 503
