# Implementation plan

## Construction

- [x] update: [pkg/ielections/impl_killer_test.go](../../../pkg/ielections/impl_killer_test.go)
  - add: Test killer rescheduled with `killDeadlineFactor` on successful CAS renewal
  - add: Test killer stays at pre-CAS schedule when CAS returns `!ok` (leadership released)
  - add: Test killer stays at pre-CAS schedule when CAS errors exhaust all retries (leadership released)
