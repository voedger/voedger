# Clarify VVM and request context naming

- URL: https://untill.atlassian.net/browse/AIR-4093
- ID: AIR-4093
- State: in-progress
- Author: Denis Gribanov
- Assignees: Denis Gribanov
- Labels: none

## Why

`ctx` is provided to `ProvideQueryProcessorStateFactory`, `ProvideAsyncActualizerStateFactory`, `ProvideSchedulerStateFactory`, `ProvideSyncActualizerStateFactory`, `ProvideMockedActualizerStateFactory`, `ProvideMockedCommandProcessorStateFactory`, `ProvideCommandProcessorStateFactory`. It is not cleare what is the context, It actually is `VVMCtx`

## What

- rename `ctx`-> `vvmCtx` where it is true
- rename `IState.Contex()` → `IState.RequestContext()`
