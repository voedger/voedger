# Clarify VVM and request context naming

- URL: https://untill.atlassian.net/browse/AIR-4093
- ID: AIR-4093
- State: in-progress
- Author: Denis Gribanov
- Assignees: Denis Gribanov
- Labels: none

## Why

`ctx` is provided to state factories (e.g. `ProvideQueryProcessorStateFactory`, `ProvideAsyncActualizerStateFactory`, `ProvideSchedulerStateFactory`, `ProvideSyncActualizerStateFactory`, `ProvideMockedActualizerStateFactory`, `ProvideMockedCommandProcessorStateFactory`, `ProvideCommandProcessorStateFactory`). The name does not clarify whether it is a VVM-lifetime context (`vvmCtx`) or a per-request context (`requestCtx`).

## What

- rename `ctx`-> `vvmCtx` where it is true
- rename `IState.Contex()` → `IState.RequestContext()`
