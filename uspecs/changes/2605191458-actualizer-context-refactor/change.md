---
registered_at: 2026-05-19T14:58:25Z
change_id: 2605191458-actualizer-context-refactor
type: refactor
scope: actualizers
baseline: e50729e8f343aceebe4189c5848147088e711df6
issue_url: https://untill.atlassian.net/browse/AIR-3972
---

# Change request: Refactor async actualizer context usage

## Why

`asyncActualizerContextState.vvmCtx` (a per-run cancellable context derived from the VVM context) is currently propagated into every place inside `asyncActualizer` that needs a `context.Context` — PLog reads, partition borrows, structured logging, pipeline error handling — while its only real responsibility is to be the cancel handle that breaks `IN10nBroker.WatchChannel`. The misleading field name and broad reuse obscure intent and tangle lifecycle: callers can't easily tell which paths need to abort with the run-scoped cancel and which should simply follow the surrounding `vvmCtx`/operation context.

## What

Narrow the surface so the actualizer only stores what is needed to break `WatchChannel` and remember the cancellation cause; let every other call site use the context already in scope. Use `context.WithCancelCause`/`context.Cause` instead of a hand-rolled struct with a mutex, and drop the vestigial `WatchChannel` invocation inside `cancelChannel` (no longer needed since AIR-1855 moved channel cleanup to `NewChannel`'s returned `channelCleanup`).

- Delete the `asyncActualizerContextState` struct (lock + err + vvmCtx + cancel + `cancelWithError`/`error()` methods) — replaced by `context.WithCancelCause`
- On `asyncActualizer`, store `n10nWatchChannelCtx context.Context` and its `cancelN10NWatchChannelCtx context.CancelCauseFunc` directly; on `asyncErrorHandler`, store only the `cancelN10NWatchChannelCtx context.CancelCauseFunc`
- Replace every `a.readCtx.cancelWithError(err)` with `a.cancelN10NWatchChannelCtx(err)`; replace `a.readCtx.error()` with `context.Cause(a.n10nWatchChannelCtx)` (filtered when the parent ctx is the one that ended the loop)
- Remove `cancelChannel(e)` entirely; the early `readPlogToTheEnd` failure path just returns the error and lets `finit`/`channelCleanup` tear down the n10n channel
- Replace every other `a.readCtx.vvmCtx` reference with the in-scope `ctx`/`vvmCtx` (`readPlogByBatches` loop, `readPlogToTheEnd`/`readPlogToOffset`'s `borrowAppPart`/`ReadPLog`, `readOffset`, `Prepare`'s retrier `OnError` logger)
- Ensure `finit` calls `cancelN10NWatchChannelCtx(nil)` so the per-run cause-context is released on every retrier iteration

## Construction

- [x] update: [actualizers/types.go](../../../pkg/processors/actualizers/types.go)
  - remove: struct `asyncActualizerContextState` and its methods `cancelWithError`/`error()`
  - remove: unused `context` and `sync` imports

- [x] update: [actualizers/async.go](../../../pkg/processors/actualizers/async.go)
  - replace on `asyncActualizer`: field `readCtx *asyncActualizerContextState` → `n10nWatchChannelCtx context.Context` + `cancelN10NWatchChannelCtx context.CancelCauseFunc`
  - replace on `asyncErrorHandler`: field `readCtx *asyncActualizerContextState` → `cancelN10NWatchChannelCtx context.CancelCauseFunc`; update the `init` initializer accordingly
  - `init(vvmCtx)`: `a.n10nWatchChannelCtx, a.cancelN10NWatchChannelCtx = context.WithCancelCause(vvmCtx)`; pass `vvmCtx` to `readOffset`
  - `finit()`: append `if a.cancelN10NWatchChannelCtx != nil { a.cancelN10NWatchChannelCtx(nil) }` so the cause-context is released even when the WatchChannel path was never reached
  - remove: method `cancelChannel(e error)` — the inner `WatchChannel` call is vestigial post-AIR-1855; channel cleanup is owned by `finit`/`channelCleanup`
  - `keepReading(ctx)`: on `readPlogToTheEnd` failure return the error directly; in the WatchChannel callback call `a.cancelN10NWatchChannelCtx(err)`; after WatchChannel returns, compute `n10nWatchCtxCause := context.Cause(a.n10nWatchChannelCtx)` and return `n10nWatchCtxCause` only when `!errors.Is(n10nWatchCtxCause, context.Cause(ctx))` (otherwise return `nil`) so parent-propagated cancellation is suppressed and does not trigger retrier `OnError` logging
  - `readPlogByBatches`/`readPlogToTheEnd`/`readPlogToOffset`/`readOffset`: replace every `a.readCtx.vvmCtx` with the in-scope `ctx`; change `readOffset` signature to `readOffset(ctx context.Context, projectorName appdef.QName)`
  - `Prepare`'s `retrierCfg.OnError`: replace `logger.ErrorCtx(a.readCtx.vvmCtx, …)` with the `vvmCtx` captured from `Prepare`'s parameter
  - `OnError`: replace `h.readCtx.cancelWithError(err)` with `h.cancelN10NWatchChannelCtx(err)`

- [x] update: [actualizers/async_test.go](../../../pkg/processors/actualizers/async_test.go)
  - add: `Test_AsynchronousActualizer_KeepReadingPropagatesReadPLogErrorAsCause` covering the path where `WatchChannel`'s notify callback triggers `readPlogToOffset`, the failure is recorded as the cause of `n10nWatchChannelCtx`, and `keepReading` returns the same error verbatim (`require.Same` between returned error and `context.Cause(act.n10nWatchChannelCtx)`)
  - add: `ctxFailingAppParts` test helper (decorator over `IAppPartitions`) that fails `WaitForBorrow` only when the incoming `ctx` is identity-equal to a configured `failOnCtx` (mutex-protected), keying off `n10nWatchChannelCtx` to avoid races with the pipeline's background `Flush`-driven borrows
  - add: `sync` import

- [x] Review
