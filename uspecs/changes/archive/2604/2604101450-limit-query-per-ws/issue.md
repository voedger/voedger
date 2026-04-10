# AIR-3542: voedger: Limit concurrent query executions per workspace to 10 (by default)

- **Type:** Sub-task
- **Status:** In Progress
- **Assignee:** d.gribanov@dev.untill.com

## Problem

During the AIR-3536 outage, 93 goroutines were simultaneously processing UPStandardWebhook requests for a single workspace, starving all other workspaces and blocking query processors entirely.

## Architecture

Implement the limiter in the router layer (`pkg/router`), not inside the query processor or VVM request handler. Why it can work:

- `bus.IResponseWriter.Write()` blocks until the router reads each item from the channel (buffer = 1, one-by-one handshake)
- The query processor calls `rowsProcessor.Close()` only after all rows are confirmed consumed by the router, then calls `respWriter.Close(err)` which closes the channel
- The router detects channel close via `for elem := range responseCh` exiting

Therefore: when `reply_v1` / `reply_v2` returns, the processor slot is provably free. Decrementing the per-WSID counter at that exact point reflects actual processor occupancy, not HTTP session duration.

## Implementation

- Add a concurrent-safe per-WSID counter map in the router (`pkg/router`)
- In `sendRequestAndReadResponse` (V2) and `RequestHandler_V1` (V1), before calling `SendRequest`: check the counter for the request WSID. If it is at or above the limit, return HTTP 503 immediately
- Increment the counter, call `SendRequest`, call `reply_v1` / `reply_v2`, then decrement (defer)
- Apply the limit to query requests only (GET in V2, resource prefix `q.` in V1). Commands have a separate protection mechanism (one channel per partition)
- The limit value is configurable via `RouterMaxQueriesPerWS` in `VVMConfig` (default 10), consistent with existing `RouterWriteTimeout`/`RouterReadTimeout`/`RouterConnectionsLimit` naming

## Testing

Reasonable tests are needed.

## Scope

voedger repository, `pkg/router`.
