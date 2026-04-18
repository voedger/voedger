# Decisions: Redirect http server internal error log to voedger logger

## Attaching reqid to forwarded http.Server.ErrorLog messages

Do not attach `reqid` to messages forwarded from `http.Server.ErrorLog`; keep per-server attributes only (`vapp`, `extension`) plus `stage=endpoint.http.error` (confidence: high).

Rationale:

- `http.Server.ErrorLog` is a single `*log.Logger` invoked by `net/http` internals without any `*http.Request` or context in scope. Most call sites have no active request at all: listener `Accept` errors, TLS handshake errors, HTTP/1 protocol parse errors, HTTP/2 framing/stream errors, keep-alive teardown, stdlib panic-serving messages
- Any `reqid` we attach at those sites would be fabricated. Using "last-seen reqid on the connection/goroutine" is actively misleading because keep-alive connections serve many requests and background goroutines do not carry our reqid — attribution to the wrong request is worse than no attribution
- Attributes stay truthful: `extension=sys._HTTPServer` + `stage=endpoint.http.error` accurately describes the source. Operators can correlate with a specific request via a close time window and, when stdlib emits it, the remote address
- Consistent with existing server-scope logs (`endpoint.listen.start`, `endpoint.shutdown`) and aligns with KISS

Alternatives:

- Per-connection reqid via `ConnContext` (confidence: low)
  - Still wrong for multi-request keep-alive connections; adds coupling and synchronization cost for near-zero benefit
- Goroutine-local `reqid` propagation (confidence: low)
  - Requires goroutine-local storage or explicit threading through stdlib internals that we do not own; fragile and intrusive
- Parse `reqid` out of stdlib log lines (confidence: low)
  - Stdlib does not include `reqid`; only remote address for some errors; no reliable signal to extract
- Log request-attributable HTTP errors separately at sites where we hold the request context (confidence: medium)
  - Complement, not replacement: keeps `ErrorLog` limited to non-request/connection-level noise while request-scoped panic recovery and handler errors continue to log with full context via `logger.ErrorCtx(reqCtx, ...)`
