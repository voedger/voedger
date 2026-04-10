# Decisions: Limit concurrent query executions per workspace

## Detecting query requests in API v2

Use `busRequest.APIPath` (the `processors.APIPath` enum) instead of the HTTP method to identify query requests (confidence: high).

Rationale: In API v2, the HTTP GET method is used by 9 different endpoint types — not just queries. The `APIPath` field is set per handler and precisely identifies what kind of request it is. The relevant values that go through the query processor are:

- `APIPath_Queries` — explicit query execution (`/queries/{pkg}.{query}`)
- `APIPath_Views` — view reads (`/views/{pkg}.{view}`)
- `APIPath_Docs` — single doc read (`/docs/{pkg}.{table}/{id}`, GET)
- `APIPath_CDocs` — collection read (`/cdocs/{pkg}.{table}`, GET)
- `APIPaths_Schema`, `APIPath_Schemas_WorkspaceRoles`, `APIPath_Schemas_WorkspaceRole` — schema reads
- Blob reads and notifications also use GET but have separate handlers

The VVM request handler (`impl_requesthandler.go:74`) currently uses `request.Method == http.MethodGet` to route to the query processor, meaning **all** GET requests go to the QP. The limiter should match the same scope — whatever the QP processes.

Alternatives:

- Check HTTP method == GET (confidence: medium)
  - Simple but semantically imprecise; would limit schema reads and blob reads equally, which may not be desired. However, since the VVM itself routes all GET to QP, this actually matches the real bottleneck scope
- Check URL path segment for `/queries/` (confidence: low)
  - Too narrow; misses views, doc reads, and schema reads which also consume QP slots and contributed to the outage

## Scope of the limiter: only `/queries/` or all QP requests

Limit all requests routed to the query processor, not just `/queries/` (confidence: high).

Rationale: The outage was caused by UPStandardWebhook requests saturating QP capacity for one workspace. Any GET request in V2 (queries, views, doc reads, schema reads) goes through the QP and can cause the same starvation. Limiting only `/queries/` would leave the door open for the same problem via other QP-bound paths.

Alternatives:

- Limit only `/queries/` endpoints (confidence: medium)
  - More surgical, but leaves views and doc reads uncapped per workspace
- Separate limits per APIPath type (confidence: low)
  - Over-engineered for the current problem; can be introduced later if needed

## Where to place the limiter check in the code

Place the check inside `sendRequestAndReadResponse` with a condition on `busRequest.APIPath` via `isQPBoundAPIPath()` (confidence: high).

Rationale: Centralizes the logic in one place — fewer code changes and less risk of missing a handler. The `busRequest.APIPath` is already set by each handler before calling `sendRequestAndReadResponse`, so the check is precise. Commands pass through too but are skipped by the `isQPBoundAPIPath` guard.

Alternatives:

- Place the check inside each specific handler function (confidence: medium)
  - More explicit per-handler, but duplicates logic across many handlers and is error-prone when new handlers are added
- Place the check in VVM request handler (confidence: low)
  - Too late; the request has already been sent through the bus, the router cannot return 503 easily

## Schema and blob reads: should they count toward the per-workspace limit

Schema reads should NOT count (confidence: high). Blob reads should NOT count (confidence: high).

Rationale: Schema endpoints (`/schemas`) don't take a WSID — they operate at the app level, so there's no workspace to limit against. Blob reads have their own separate handler (`blobRequestHandler`) and don't go through `sendRequestAndReadResponse` the same way. Both are lightweight and weren't part of the outage scenario.

Alternatives:

- Include all GET endpoints uniformly (confidence: low)
  - Schema endpoints lack WSID, making per-workspace limiting impossible without redesign

## Notification subscribe-and-watch: should it count toward the limit

Notifications should NOT count toward the per-workspace query limit (confidence: high).

Rationale: The subscribe-and-watch endpoint is long-lived (SSE/WebSocket style) and has `IsN10N = true` set. It uses a completely different processing path. Including it would quickly exhaust the limit and block normal queries.

Alternatives:

- Include notifications with a separate higher limit (confidence: low)
  - Adds complexity for no clear benefit; notifications weren't part of the outage
