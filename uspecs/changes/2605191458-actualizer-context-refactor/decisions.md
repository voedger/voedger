# Decisions

## Uncertainty: shape of the read-cancel state

Decision: drop the dedicated struct entirely and use `context.WithCancelCause`/`context.Cause` on `n10nWatchChannelCtx`; store only the `context.CancelCauseFunc` on `asyncActualizer` and `asyncErrorHandler`

- Pros: removes the manual `sync.Mutex` + `err` field; first-cause-wins semantics are guaranteed by the standard library; no awkward `a.readCtx.cancelWithError(err)` / `a.readCtx.error()` call sites; aligns with Go 1.20+ idiom
- Cons: callers must filter `context.Cause` against `ctx.Err()` to preserve the existing "return nil on parent shutdown" behavior of `asyncActualizerContextState.error()`; `finit` must call `cancelN10NWatchChannelCtx(nil)` so the per-run cause-context registered on `vvmCtx` is released on every retrier iteration
- Field naming: `n10nWatchChannelCtx context.Context` + `cancelN10NWatchChannelCtx context.CancelCauseFunc` (paired names make the cancel-handle's target explicit at every call site)
- Confidence: high
- Supersedes: the earlier `onReadError` rename decision (the struct is now removed)

## Uncertainty: keep `cancelChannel`'s inner `WatchChannel` call?

Decision: remove `cancelChannel` entirely; the early `readPlogToTheEnd` failure path returns the error directly and `finit`/`channelCleanup` tears down the n10n channel

- Pros: the inner `WatchChannel` call with an already-cancelled ctx has been dead code since AIR-1855 (c9625f6a1) moved cleanup from `WatchChannel`'s defer into the `channelCleanup` func returned by `NewChannel`; today it only toggles `channel.watching` from false→true→false via the deferred `Store(false)` and does no useful work
- Cons: any future change that puts channel-side effects back into `WatchChannel` would need to re-add an explicit cleanup invocation; mitigated by the fact that `channelCleanup` is the single, explicit cleanup path
- Confidence: high

## Uncertainty: better name for `asyncActualizerContextState` (superseded)

Decision (superseded): `onReadError` — kept as historical record; the struct itself was later removed in favor of `context.WithCancelCause`

- Pros: named the struct after the event it handled
- Cons: noun-as-handler naming; method names read awkwardly at call sites
- Confidence: user-provided

Alternatives:

1. `readCancel`
   - Pros: minimal change to surrounding code; mirrors current `readCtx` naming so call sites stay readable
   - Cons: "Cancel" alone doesn't convey the stored cause; still feels like a context-ish thing
   - Confidence: medium
2. `readLoopTerminator`
   - Pros: states exactly what it does (terminates the read loop, which includes breaking `WatchChannel`); matches the file's existing vocabulary (`readPlogToTheEnd`, `keepReading`)
   - Cons: "terminator" sounds heavy; ties name to "loop" while the struct only stores cancel+err
   - Confidence: high
3. `runCanceller`
   - Pros: scopes intent to one actualizer run; pairs naturally with the existing `cancelWithError`/`cancel`/`error()` methods
   - Cons: generic; doesn't hint that the cause is preserved
   - Confidence: high
4. `watchChannelBreaker`
   - Pros: matches the issue wording literally; obvious why the struct exists
   - Cons: under-sells responsibility — it also tears down PLog reads and the pipeline indirectly via the cancel; ties name to one downstream consumer
   - Confidence: medium
5. `abortSignal`
   - Pros: short; reads naturally at call sites (`a.abort.cancelWithError(err)`)
   - Cons: "signal" overlaps with OS signal handling vocabulary used elsewhere in the codebase
   - Confidence: medium
6. `cancelCause`
   - Pros: aligns with Go 1.20+ `context.WithCancelCause` vocabulary
   - Cons: invites confusion with the std-lib helper even though the struct doesn't wrap it; risks future refactor pressure to actually use `context.WithCancelCause`
   - Confidence: low
