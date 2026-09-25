---
change_id: 2609250858-remove-100-cud-request-limit
type: feat
issue_url: https://untill.atlassian.net/browse/AIR-4985
domains: [prod]
scope: [apps, routing]
---

# Change request: Remove fixed CUD count limit from requests

Refs:

- [AIR-4985: voedger: eliminate limitation of 100 cuds per requests, rely on request body size limitation instead](./issue-AIR-4985.md)

## Why

Command requests are rejected when they contain more than 100 CUD operations even when their payload is within the request body size limit. Removing the fixed count limit allows clients to submit larger valid transactions while retaining the body-size safeguard that bounds request resource use.

## What

- In the production apps context, command requests are not rejected solely because they contain more than 100 CUD operations.
- Command requests with more than 100 CUD operations are accepted when they satisfy the request body size limit and all existing command and CUD validation rules.
- In the production routing context, requests that exceed the request body size limit continue to be rejected.

## How

Decisions:

- Use the routing ingress limit on actual request-body bytes as the sole generic resource bound for external command payloads; do not replace the command processor's fixed CUD-count check with another count-based limit.
- Keep larger CUD arrays in the existing command transaction and apply the same per-CUD parsing, authorization, validation, ID generation, persistence, and actualization behavior used for smaller arrays.
- Decouple the command and actualizer intent limit from the obsolete request CUD maximum while preserving its current value of 1,000 as an independent internal safety limit.
- Verify both boundaries: command processing accepts a valid payload above the former CUD count limit when it fits within the body limit, and routing still rejects oversized payloads before dispatch.

Assumptions:

- None

Out of scope:

- Changing the 200,000-byte function request body limit or its HTTP 413 response contract.
- Changing projector, scheduler, or application-specific intent and business limits.

References:

- [command CUD parsing and validation pipeline](../../../../../pkg/processors/command/impl.go)
- [command processing and transaction pipeline](../../../../../pkg/processors/command/provide.go)
- [shared request and intent limits](../../../../../pkg/sys/builtin/consts.go)
- [actualizer intent limit configuration](../../../../../pkg/processors/actualizers/consts.go)
- [function request body enforcement](../../../../../pkg/router/impl_validation.go)
- [request body boundary coverage](../../../../../pkg/router/impl_test.go)
- [production routing ingress contract](../../../../specs/prod/routing/arch-ingress.md)

## Technical design

- [x] update: [prod/apps/arch-processing.md](../../../../specs/prod/apps/arch-processing.md)
  - update: command CUD parsing and validation to state that apps processing does not impose a fixed CUD-count limit and processes every CUD present in the request
  - clarify: routing ingress owns the generic external request-size boundary, while the existing per-CUD authorization, validation, and transactional processing remain in apps

## Construction

- [x] update: [sys/it/impl_cud_test.go](../../../../../pkg/sys/it/impl_cud_test.go)
  - add: end-to-end HTTP coverage posting 300 compact CUDs within the request body limit and verifying that the full transaction is persisted

- [x] update: [actualizers/consts.go](../../../../../pkg/processors/actualizers/consts.go)
  - replace: the request-limit-derived `DefaultIntentsLimit` with an independent value of 1,000, preserving current command and actualizer behavior
  - remove: the now-unused dependency on the built-in request CUD limit

- [x] update: [command/impl.go](../../../../../pkg/processors/command/impl.go)
  - remove: the fixed `MaxCUDs` rejection from command CUD parsing so every submitted CUD continues through the existing parsing and validation pipeline

- [x] update: [builtin/consts.go](../../../../../pkg/sys/builtin/consts.go)
  - remove: the obsolete `MaxCUDs` constant after its request and actualizer responsibilities are eliminated or decoupled
