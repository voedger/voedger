# Implementation plan

## Technical design

- [x] update: [apps/vvm-orch--arch.md](../../specs/prod/apps/vvm-orch--arch.md)
  - update: Leadership renewal interval from TTL/2 to TTL/4 with retry logic
  - update: maintainLeadership description to reflect retry-on-error behavior within each interval
  - update: Timing relationships diagram

## Construction

- [x] update: [pkg/ielections/impl.go](../../../pkg/ielections/impl.go)
  - update: Change tickerInterval from TTL/2 to TTL/4
  - update: Add renewWithRetry function with retry loop on CompareAndSwap error (retry every second during interval)
  - update: Fail fast on !ok (value mismatch)
  - update: Check context on each retry iteration
  - update: Rename parameter `ttlSeconds` → `leadershipDurationSeconds`
  - update: Move and rename constant `tickerIntervalDivisor` → `renewalsPerLeadershipDur`

- [x] add: [pkg/ielections/consts.go](../../../pkg/ielections/consts.go)
  - add: Extracted `renewalsPerLeadershipDur` constant

- [x] update: [pkg/ielections/impl_test.go](../../../pkg/ielections/impl_test.go)
  - add: TestTransientErrorRecovery — CAS retried until no error, leadership retained
  - add: TestAcuireLeadershipFailureAfterCompareAndSwapError — persistent CAS error leads to leadership loss after deadline
  - add: TestCleanupDuringCompareAndSwapRetries — cleanup during CAS retries cancels leadership context
