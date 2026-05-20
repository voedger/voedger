---
registered_at: 2026-05-20T21:16:29Z
change_id: 2605202116-actualizer-context-watchchannel
type: refactor
scope: apps
baseline: c27f74d7bd4b234f964a9724a0c2ae54d69aadb4
issue_url: https://untill.atlassian.net/browse/AIR-3972
archived_at: 2026-05-20T21:27:38Z
---

# Change request: Refactor actualizer context usage

## Why

`asyncActualizerContextState.vvmCtx` is used across the actualizer context but is only needed to break `WatchChannel`. Keeping it in the broad context state exposes an implementation detail to code paths that should not depend on it.

## What

Deliver an internal refactor with no behavior change: actualizer processing must preserve the existing `WatchChannel` lifecycle and cancellation behavior while narrowing the scope of the context value used to break it.

- Remove `vvmCtx` from the broad async actualizer context state.

- Introduce a thin-scoped mechanism dedicated to breaking `WatchChannel`.

## How

Decisions:

- Keep `asyncActualizerContextState` responsible for synchronized error capture and cancellation signaling, but remove direct ownership of the VVM-derived context from that shared state.

- Create a local `watchCtx` in the async actualizer read flow and pass it only to `IN10nBroker.WatchChannel`, so the cancellable context exists only where it is needed to unblock waiting.

- Keep PLog reads, app partition borrowing, state factory creation, pipeline construction, and logging on the parent actualizer context.

- Preserve existing async actualizer retry, PLog reading, pipeline error handling, channel cleanup, and offset persistence behavior.

Out of scope:

- Changing `IN10nBroker.WatchChannel` API semantics.
- Changing async projector execution, batching, or flush policy.

References:

- [async actualizer context state](../../../../../pkg/processors/actualizers/types.go)
- [async actualizer processing flow](../../../../../pkg/processors/actualizers/async.go)
- [notification broker contract](../../../../../pkg/in10n/interface.go)
- [async actualizer tests](../../../../../pkg/processors/actualizers/async_test.go)

## Construction

- [x] update: [actualizers/async_test.go](../../../../../pkg/processors/actualizers/async_test.go)
  - add or update regression coverage proving WatchChannel can be unblocked through the local watch context without storing the VVM-derived context in `asyncActualizerContextState`
  - verify async actualizer error and retry behavior still preserves PLog reading, channel cleanup, and offset persistence behavior

- [x] update: [actualizers/types.go](../../../../../pkg/processors/actualizers/types.go)
  - remove `vvmCtx` from `asyncActualizerContextState`
  - keep synchronized error storage and cancellation signaling needed by the async error handler

- [x] update: [actualizers/async.go](../../../../../pkg/processors/actualizers/async.go)
  - create the local `watchCtx` in the async actualizer read flow and pass it only to `IN10nBroker.WatchChannel`
  - route PLog reads, app partition borrowing, state factory creation, pipeline construction, and logging through the parent actualizer context
  - ensure error paths that currently call `cancelWithError` still unblock `WatchChannel` and return the captured error to the retry loop
