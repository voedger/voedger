# Implementation plan: Simplify IState reuse in processors

## Construction

- [x] update: [pkg/processors/command/types.go](../../pkg/processors/command/types.go)
  - remove: `hostStateProvider` struct, its 12 getter methods, `newHostStateProvider` func, and `get` method
  - add: `reusableHostState` struct with a `*cmdWorkpiece` pointer field and a `bind(*cmdWorkpiece)` method
  - add: `newReusableHostState` func that creates the struct and passes closures reading from the bound workpiece pointer to `ProvideCommandProcessorStateFactory`
  - rename: `cmdWorkpiece.hostStateProvider` field to `hostState`
- [x] update: [pkg/processors/command/impl.go](../../pkg/processors/command/impl.go)
  - update: `getHostState` to call `hostState.bind()` instead of `get()` with 12 arguments, then `ClearIntents` and assign `State`/`Intents`
  - update: `checkResponseIntent` to use `hostState.state` instead of `hostStateProvider.state`
- [x] update: [pkg/processors/command/provide.go](../../pkg/processors/command/provide.go)
  - update: replace `newHostStateProvider` call with `newReusableHostState`
  - update: `cmdWorkpiece` initialization to use `hostState` field
- [x] no changes needed: [pkg/processors/command/impl_test.go](../../pkg/processors/command/impl_test.go)
  - no references to `hostStateProvider` found in tests
