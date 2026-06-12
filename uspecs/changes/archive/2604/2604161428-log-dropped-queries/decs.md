# Decisions: Log dropped queries per workspace limit

## Aggregation key and data structure

Use `map[rejectionKey]*rejectionCounter` guarded by `sync.Mutex` with `rejectionCounter{count int64, logCtxFromLastQuery context.Context}`. Add `lastLoggedAt int64`, `mu sync.Mutex`, and `iTime timeu.ITime` fields to `wsQueryLimiter` (confidence: high).

Rationale: per-[wsid, extension] aggregation matches the issue requirement. A mutex-guarded map eliminates race conditions between counter increment, timestamp update, and purge — which were problematic with atomics + `sync.Map`. `tryFlush` and `flushAll` swap the map under the lock and log outside it to minimize lock hold time.

Alternatives:

- `sync.Map` with atomics (confidence: medium)
  - Race conditions between purge and concurrent `onQueryDrop`: delete can remove an entry that a concurrent goroutine is about to update
- Single global counter (confidence: low)
  - Loses per-key granularity required by the issue

## Flush trigger mechanism

On every query request (accepted or rejected), acquire mutex and check `lastLoggedAt` via `iTime.Now()`. If non-zero and 10 seconds have passed, swap the rejections map under the lock, update `lastLoggedAt`, release the lock, then log outside the lock (confidence: high).

Rationale: no timers, no goroutines, no channels. The mutex is held only briefly (map swap + timestamp update). Logging happens outside the lock. `lastLoggedAt` is set to `iTime.Now().UnixNano()` on the first rejection and updated to current time after each flush. Using `ITime` enables mocked time in tests.

Alternatives:

- CAS-guarded `time.AfterFunc(10s)` per key (confidence: medium)
  - One timer per key; more complex lifecycle; race conditions on purge
- Permanent background ticker (confidence: medium)
  - Wastes resources when idle

## Context storage

Store request context directly in `rejectionCounter.logCtxFromLastQuery` on each rejection (confidence: high).

Rationale: the issue requires "LogCtx from the last query." The request context carries structured log attributes (WSID, app, extension, reqID) set by `withLogAttribs`. Storing on each rejection ensures the latest context is always used. The context is stored under mutex, so no atomic wrapper is needed.

Alternatives:

- Store only on first rejection (confidence: medium)
  - Simpler, but uses a stale request's context; issue says "last query"
- Build a fresh context with only the key fields (confidence: low)
  - Duplicates attribute logic already in `withLogAttribs`

## Log message format

Use `fmt.Sprintf("droppedInLast10Seconds=%d", n)` (confidence: high).

Rationale: `droppedInLast10Seconds` matches the issue specification. Comma-separated key=value format is consistent with existing structured log messages in the router.

Alternatives:

- Include limiter map size: `qpLimiterSize=X` (confidence: medium)
  - Extra info, but `maxQPerWS` is more actionable for operators
- Include wsid in message text (confidence: low)
  - Already present as structured attribute in the context

## Flush on shutdown and purging

Call `flushAll()` in `routerService.Stop()` before stopping the HTTP server. Swap the rejections map under the lock, then log outside it (confidence: high).

Rationale: `Stop()` is guaranteed to be called on VVM shutdown. Flushing before `httpServer.Stop()` ensures all pending rejections are logged. Map swap under mutex is race-free — no concurrent goroutine can access stale entries.

Alternatives:

- `context.AfterFunc(ctx, flushAll)` on VVM context cancellation (confidence: medium)
  - Works but adds dependency on context wiring; `Stop()` is simpler and already available
- No purging, rely on GC (confidence: medium)
  - Entries are bounded by distinct [wsid, extension] pairs; acceptable but explicit cleanup is cleaner
