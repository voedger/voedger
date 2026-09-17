---
change_id: 2609171303-limit-router-content-length
type: feat
issue_url: https://untill.atlassian.net/browse/AIR-1208
domains: [prod]
scope: [routing]
breaking: true
---

# Change request: Query and command content limit

Refs:

- [AIR-1208: voedger: router: limit q/c content length: 200K](./issue-AIR-1208.md)

## Why

Supporting bills with up to 300 items requires larger command and query requests while keeping the HTTP boundary protected from unbounded request content. A defined 200K cap at the shared buffered-request boundary gives clients enough room for the larger bills and bounds memory consumed by routing ingress.

## What

- The `prod` routing context caps every request body read by the shared buffered function-request validation boundary at 200K. This includes API v1 function calls and the non-streaming API v2 docs, cdocs, commands, queries, views, schemas and roles, auth, user, change-password, and device endpoints.
- Buffered request content at or below the cap continues through the existing request validation and dispatch flow.
- Buffered request content above the cap is rejected at the HTTP boundary before endpoint-specific processing or dispatch.

## How

Decisions:

- Enforce the cap once in `withValidateForFuncs`, the shared buffered function-request validation boundary used by API v1 and non-streaming API v2 handlers; keep BLOB and N10N request-body paths on their existing flows.
- Use the Go standard library's bounded request-body reader and count bytes actually read instead of trusting `Content-Length`, so missing, chunked, or inaccurate headers cannot bypass the cap.
- Map a body-limit overflow to HTTP `413 Request Entity Too Large` with the message `request body size limit exceeded` through the existing JSON and CORS response path; retain `400 Bad Request` for other validation failures.
- Define the limit as a fixed router-wide value of 200,000 bytes rather than an operator setting, giving public and localhost-admin function requests the same boundary.
- Verify the shared boundary with router-level HTTP coverage across command and query routes in both API generations, prove that rejected requests are not forwarded to application processing, and exercise the exact boundary and overflow through VIT.

Assumptions:

- The ticket's 200K denotes 200,000 bytes, consistent with the decimal-byte calculations used by the related payload-size research.

Out of scope:

- Limiting BLOB transfers, notification payloads, reverse-proxied traffic, or response bodies.
- POS-side payload preflight checks and end-user messaging for oversized requests.

References (internal):

- [shared request validation boundary](../../../../../pkg/router/impl_validation.go)
- [API v1 routing and CORS boundary](../../../../../pkg/router/impl_http.go)
- [API v2 request dispatch](../../../../../pkg/router/impl_apiv2.go)
- [router error response conventions](../../../../../pkg/router/utils.go)
- [router HTTP behavior coverage](../../../../../pkg/router/impl_test.go)
- [VIT HTTP conventions coverage](../../../../../pkg/sys/it/impl_httpconventions_test.go)
- [routing ingress architecture](../../../../../uspecs/specs/prod/routing/arch-ingress.md)

References (external):

- [Go bounded request-body reader](https://pkg.go.dev/net/http#MaxBytesReader)
- [parent payload-error contract](https://untill.atlassian.net/browse/AIR-1202)
- [related payload-size research](https://untill.atlassian.net/browse/AIR-4896)

## Technical design

- [x] update: [prod/routing/arch-ingress.md](../../../../specs/prod/routing/arch-ingress.md)
  - update: the request-validation boundary to document the fixed 200,000-byte cap for every body buffered by `withValidateForFuncs` across API v1 and API v2
  - add: oversized-request handling that counts bytes read independently of `Content-Length`, returns HTTP 413 with `request body size limit exceeded`, and stops before endpoint-specific processing or application dispatch
  - update: the API v1 and API v2 dispatch flows to include the request-body limit

## Construction

- [x] update: [router/impl_test.go](../../../../../pkg/router/impl_test.go)
  - add: router-level HTTP coverage for command and query requests through API v1 and API v2 at and above the 200,000-byte boundary
  - verify: actual bytes are limited when `Content-Length` is missing or inaccurate and when the request uses chunked transfer encoding
  - verify: a body of exactly 200,000 bytes is forwarded unchanged, while a longer body returns HTTP 413 with `{"status":413,"message":"request body size limit exceeded"}` and never reaches the application request handler
  - verify: the shared limit applies to both public and localhost-admin function requests

- [x] update: [sys/it/impl_httpconventions_test.go](../../../../../pkg/sys/it/impl_httpconventions_test.go)
  - add: VIT coverage proving that a valid function request of exactly 200,000 bytes is processed and a 200,001-byte request returns HTTP 413 with `request body size limit exceeded`

- [x] update: [router/consts.go](../../../../../pkg/router/consts.go)
  - add: private constants for the fixed 200,000-byte function-request body limit and the client-facing overflow message

- [x] update: [router/impl_validation.go](../../../../../pkg/router/impl_validation.go)
  - update: the shared function-request wrapper to bound the actual request-body stream with the Go standard-library request-body limiter before existing validation reads it
  - update: centralized validation error handling to recognize body-limit overflow and return HTTP 413 with `request body size limit exceeded`; retain HTTP 400 for all other validation failures
  - preserve: existing body handling for BLOB, N10N, and reverse-proxy traffic
