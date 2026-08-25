---
change_id: 2608241226-partition-borrow-retry-error
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-4768
domains: [prod]
scope: [apps]
---

# Change request: Correct partition borrow retry error handling

Refs:

- [AIR-4768: voedger, appparts: fix wrong error checked in partition borrow retry handler](./issue-AIR-4768.md)

## Why

Application processors can hang when their extension engine pool is exhausted because partition borrowing does not recognize the terminal engine-availability error. The retry decision must use the operation error so engine exhaustion is reported instead of retried indefinitely.

### Why existing tests passed

The existing test suite did not expose the incorrect retry
classification because:

- regression coverage asserted the same reversed classification as the
  implementation, so it confirmed the defect instead of detecting it
- the deployment-test runner handled a surfaced
  `ErrNotAvailableEngines` by immediately continuing its own loop,
  masking the behavior at the `WaitForBorrow` boundary
- VIT normally provisions engine pools large enough that integration
  tests rarely exhaust them
- query processor integration tests call `Borrow` directly and therefore
  do not exercise the `WaitForBorrow` retry policy

## What

Symptom: A processor waits indefinitely when partition borrowing fails because no extension engine is available.

```text
processor requests an application partition through WaitForBorrow
      |
      v
Borrow returns ErrNotAvailableEngines
      |
      v
partBorrowRetryCfg.OnError receives opErr
      |
      v
OnError checks named return variable err instead of opErr   <-- fault
      |
      v
OnError requests another retry
      |
      v
WaitForBorrow retries indefinitely   (symptom)
```

Corrected behavior: `ErrNotAvailableEngines` aborts partition borrowing and is returned to the processor, while other transient borrow errors remain retryable.

## How

Decisions:

- Keep borrow-error classification in the shared application-partition retry policy so command, query, actualizer, and scheduler processors receive consistent behavior without processor-specific handling
- Continue classifying engine exhaustion with `errors.Is` against the public sentinel so processor-specific wrapped errors retain their identity and diagnostics
- Verify the retry contract at the application-partition boundary across every processor kind rather than duplicating coverage in individual processor packages

Assumptions:

- None

Out of scope:

- Preventing scheduler or job re-entry
- Changing extension-engine pool sizes or partition borrow retry delays

References:

- [shared partition borrowing and retry policy](../../../../../pkg/appparts/impl.go)
- [processor-specific engine exhaustion errors](../../../../../pkg/appparts/errors.go)
- [engine borrow and release lifecycle](../../../../../pkg/appparts/impl_app.go)
- [existing application-partition borrowing coverage](../../../../../pkg/appparts/impl_test.go)

## Construction

- [x] create: [appparts/impl_internal_test.go](../../../../../pkg/appparts/impl_internal_test.go)
  - provide package-internal regression coverage for the application-partition borrow retry policy
  - verify that every processor-specific wrapped `ErrNotAvailableEngines` error enables retry without an abort error
  - verify that a non-engine-availability error disables retry and is returned as the abort error

- [x] update: [appparts/impl.go](../../../../../pkg/appparts/impl.go)
  - classify the operation error received by `partBorrowRetryCfg.OnError` instead of its named return variable
  - retry engine-exhaustion errors and abort with the operation error for all other borrow failures
