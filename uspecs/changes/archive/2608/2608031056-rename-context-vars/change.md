---
change_id: 2608031015-rename-context-vars
type: refactor
issue_url: https://untill.atlassian.net/browse/AIR-4093
domains: [prod]
scope: [apps, storage, extensions]
breaking: true
---

# Change request: Explicit context variable names

Refs:

- [AIR-4093: Clarify VVM and request context naming in state APIs](./issue-AIR-4093.md)

## Why

Generic `ctx` names obscure whether processor state and factory code receives a VVM-lifetime context or a single-request context. Naming each context by its actual lifetime makes cancellation ownership and propagation easier to understand and maintain.

## What

- VVM-lifetime context parameters and variables are consistently named `vvmCtx`.
- Individual-request context parameters and variables are consistently named `requestCtx`.
- The `IState` context accessor is renamed as requested; other Go APIs, context values, cancellation behavior, processor behavior, and externally observable output remain unchanged.

## How

Decisions:

- Classify context names from the lifetime established at each factory boundary: query processor state uses `requestCtx`, while command processor, actualizer, and scheduler state use `vvmCtx`; mocked state follows the production role it represents.
- Apply the lifetime-specific name consistently through role-specific factory types, provider implementations, and direct call sites; use a neutral `stateCtx` name only inside shared state or test-helper code that can receive either lifetime.
- Rename `istructs.IState.Context()` to `RequestContext()` as one coordinated source-contract change across the interface, implementations, mocks, and callers; do not retain the ambiguous accessor as a compatibility alias.
- Preserve the context instances, values, parent relationships, and cancellation paths currently passed through the system; only identifiers and the requested accessor contract change.
- Use compile-time interface conformance plus targeted state and processor tests to verify that every implementation and consumer migrates together.

Assumptions:

- No out-of-repository `istructs.IState` implementation or caller requires a compatibility period for the accessor rename.

Out of scope:

- Renaming generic `ctx` identifiers outside the state-factory and `IState` context boundary.
- Introducing new child contexts or changing context ownership and cancellation behavior.

References:

- [state factory context contracts](../../../../../pkg/state/types.go)
- [public state context contract](../../../../../pkg/istructs/recources-types.go)
- [shared host-state context storage](../../../../../pkg/state/stateprovide/impl_host_state.go)
- [request-scoped query state creation](../../../../../pkg/processors/query/impl.go)
- [VVM-scoped command state lifecycle](../../../../../pkg/processors/command/provide.go)
- [VVM-scoped actualizer state creation](../../../../../pkg/processors/actualizers/async.go)
- [VVM-scoped scheduler state creation](../../../../../pkg/processors/schedulers/impl_scheduler.go)

## Construction

### Tests

- [x] update: [stateprovide/impl_host_state_test.go](../../../../../pkg/state/stateprovide/impl_host_state_test.go)
  - add: verify that `RequestContext()` returns the exact context supplied when host state is created
  - preserve: existing host-state and factory coverage while compiling against the renamed accessor

- [x] update: [appparts/impl_test.go](../../../../../pkg/appparts/impl_test.go)
  - rename: actualizer and scheduler runner mock parameters from `ctx` to `vvmCtx` to reflect the lifecycle contract under test

- [x] update: [schedulers/impl_test.go](../../../../../pkg/processors/schedulers/impl_test.go)
  - rename: scheduler lifecycle context variables to `vvmCtx` without changing cancellation assertions or timing behavior

### Contracts and shared state

- [x] update: [istructs/recources-types.go](../../../../../pkg/istructs/recources-types.go)
  - rename: the breaking `IState.Context()` method to `IState.RequestContext()`
  - preserve: the returned `context.Context` type and all other state operations

- [x] update: [state/types.go](../../../../../pkg/state/types.go)
  - rename: `QueryProcessorStateFactory` context parameter to `requestCtx`
  - rename: command processor, actualizer, scheduler, and mocked state factory context parameters to `vvmCtx`
  - preserve: factory function signatures, parameter order, and return types

- [x] update: [actualizers/interface.go](../../../../../pkg/processors/actualizers/interface.go)
  - rename: exported `SyncActualizerConf.Ctx` field to `VvmCtx`

- [x] update: [appparts/interface.go](../../../../../pkg/appparts/interface.go)
  - name: actualizer and scheduler runner lifecycle context parameters `vvmCtx`

- [x] update: [coreutils/mock.go](../../../../../pkg/coreutils/mock.go)
  - rename: `MockState.Context()` to `RequestContext()` so the shared mock continues to implement `istructs.IState`
  - preserve: existing testify mock invocation behavior

- [x] update: [stateprovide/impl_host_state.go](../../../../../pkg/state/stateprovide/impl_host_state.go)
  - rename: `Context()` to `RequestContext()` and return the same stored context instance
  - rename: shared internal context storage and constructor input to neutral `stateCtx`, since host state serves both request- and VVM-scoped factories

### State providers

- [x] update: [stateprovide/impl_query_processor_state.go](../../../../../pkg/state/stateprovide/impl_query_processor_state.go)
  - rename: the factory context parameter and its uses to `requestCtx`

- [x] update: [stateprovide/impl_command_processor_state.go](../../../../../pkg/state/stateprovide/impl_command_processor_state.go)
  - rename: the factory context parameter and its uses to `vvmCtx`

- [x] update: [stateprovide/impl_async_actualizer_state.go](../../../../../pkg/state/stateprovide/impl_async_actualizer_state.go)
  - rename: the factory context parameter and its uses to `vvmCtx`

- [x] update: [stateprovide/impl_sync_actualizer_state.go](../../../../../pkg/state/stateprovide/impl_sync_actualizer_state.go)
  - rename: the factory context parameter and its uses to `vvmCtx`

- [x] update: [stateprovide/impl_scheduler_state.go](../../../../../pkg/state/stateprovide/impl_scheduler_state.go)
  - rename: the factory context parameter and its uses to `vvmCtx`

- [x] update: [stateprovide/impl_mocked_state.go](../../../../../pkg/state/stateprovide/impl_mocked_state.go)
  - rename: mocked command and actualizer state context parameters to `vvmCtx`
  - remove: the redundant unused `MockedState.ctx` field
  - preserve: the same context instance in each mocked host state

### Processor and state consumers

- [x] update: [command/types.go](../../../../../pkg/processors/command/types.go)
  - rename: the reusable command host-state constructor context to `vvmCtx`
  - preserve: reuse of one host state across command requests

- [x] update: [teststate/impl.go](../../../../../pkg/state/teststate/impl.go)
  - rename: the shared test-state context field and uses to neutral `stateCtx`, since the helper constructs actualizer, command, or query state according to processor kind

- [x] update: [teststate/impl_new.go](../../../../../pkg/state/teststate/impl_new.go)
  - update: command and projector test-state construction to use the renamed neutral `stateCtx` field

- [x] update: [actualizers/provide.go](../../../../../pkg/processors/actualizers/provide.go)
  - update: initialize `SyncActualizerConf.VvmCtx` after the exported field rename

- [x] update: [actualizers/impl.go](../../../../../pkg/processors/actualizers/impl.go)
  - update: use `SyncActualizerConf.VvmCtx` for sync pipelines and state creation

- [x] update: [appparts/const_null.go](../../../../../pkg/appparts/const_null.go)
  - rename: null runner lifecycle context parameters and uses to `vvmCtx`

- [x] update: [schedulers/impl_schedulers.go](../../../../../pkg/processors/schedulers/impl_schedulers.go)
  - rename: the scheduler runner lifecycle context and its uses to `vvmCtx`

- [x] update: [schedulers/impl_scheduler.go](../../../../../pkg/processors/schedulers/impl_scheduler.go)
  - rename: the scheduler lifecycle field, `Run` parameter, and dependent uses to `vvmCtx`
  - preserve: retry, cancellation, borrowing, invocation, and shutdown behavior

- [x] update: [cluster/impl_vsqlupdate.go](../../../../../pkg/cluster/impl_vsqlupdate.go)
  - replace: the removed state `Context()` accessor with `RequestContext()` when dispatching VSQL updates

- [x] update: [workspace/impl.go](../../../../../pkg/sys/workspace/impl.go)
  - replace: the removed state `Context()` accessor with `RequestContext()` when allocating a workspace ID
