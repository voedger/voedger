# Decisions

## Uncertainty: where should the new thin-scoped WatchChannel breaker live?

Decision: Local `watchCtx` in async actualizer read flow

- Pros: smallest refactor; keeps the context scoped to `WatchChannel`; avoids adding new shared state
- Cons: cancellation has to be threaded carefully from error paths that currently call `readCtx.cancelWithError`
- Confidence: high

Alternatives:

1. Dedicated small struct owned by `asyncActualizer`
   - Pros: makes the breaker explicit and reusable across `keepReading` and `cancelChannel`
   - Cons: adds another lifecycle object that must be reset during retry/finit
   - Confidence: medium
2. Extend `asyncActualizerContextState` with a narrower cancel-only field
   - Pros: minimal call-site churn; keeps existing error-handler wiring mostly intact
   - Cons: only partially solves the broad-state coupling because WatchChannel control still lives in shared read state
   - Confidence: low

## Uncertainty: which operations should keep using the parent actualizer context instead of the local `watchCtx`?

Decision: Only `WatchChannel` uses `watchCtx`; PLog reads, app partition borrowing, state factory, pipeline, and logging keep using the parent actualizer context

- Pros: preserves existing cancellation/logging behavior everywhere except the dedicated WatchChannel breaker; smallest behavior risk
- Cons: implementation must audit every current `a.readCtx.vvmCtx` usage and replace it deliberately
- Confidence: high

Alternatives:

1. Use `watchCtx` for `WatchChannel` and callback-triggered PLog reads
   - Pros: aligns reads triggered by notifications with the watch lifecycle
   - Cons: may cancel PLog reads earlier than the current behavior and blur the narrow scope goal
   - Confidence: low
2. Replace most `a.readCtx.vvmCtx` usages with `watchCtx`
   - Pros: simple mechanical replacement in the current read loop
   - Cons: changes broader actualizer cancellation behavior and contradicts the "thin-scoped" requirement
   - Confidence: low
