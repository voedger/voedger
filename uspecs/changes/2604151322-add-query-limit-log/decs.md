# Decisions: Add log when query execs per workspace limit is reached

## Log level

Use `Warning` level (confidence: high).

Rationale: the limit being reached is an operational signal that a workspace is under heavy query load, but it is not an application error — it is the system working as designed. `Warning` is the correct severity for "attention needed but not broken" conditions.

Alternatives:

- `Info` (confidence: medium)
  - Too easy to miss in production; limit exhaustion is noteworthy enough to warrant `Warning`
- `Error` (confidence: low)
  - Overstates severity; this is expected throttling behavior, not a fault

## Log stage attribute

Use `routing.qp.limit` (confidence: high).

Rationale: follows the existing `routing.*` naming convention in the router package (e.g. `routing.accepted`, `routing.latency1`, `routing.send2vvm.error`, `routing.response.error`). The `qp` segment aligns with the existing `routing.qpLimiterSize` pattern and clearly identifies the query-per-workspace limiter.

Alternatives:

- `routing.qpLimiterExceeded` (confidence: medium)
  - Verbose; the shorter `routing.qp.limit` is sufficient and more consistent with dot-separated stages
- `routing.query.limit` (confidence: medium)
  - Less specific; `qp` already established in `routing.qpLimiterSize`

## Logging point — before or after context enrichment

Move the limiter check after `withLogAttribs` so the log message carries structured context (WSID, appQName, extension, reqID) (confidence: high).

Rationale: currently in `sendRequestAndReadResponse` (V2) and `RequestHandler_V1` (V1), the limiter check happens before `withLogAttribs` enriches the context. Without the enriched context the log line would lack WSID/app/extension attributes, reducing its diagnostic value. Reordering to call `withLogAttribs` first, then check the limiter, gives the warning full structured context at negligible cost.

Alternatives:

- Log with manually constructed message including WSID (confidence: medium)
  - Duplicates the attribute logic already in `withLogAttribs`; inconsistent with the structured logging pattern used throughout the router
- Keep current order and log without context (confidence: low)
  - Loses the primary diagnostic value — knowing which workspace hit the limit

## Log message content

Use empty msg
