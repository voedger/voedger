# Implementation plan

## Technical design

- [x] update: [apps/vvm-orch--arch.md](../../../../specs/prod/apps/vvm-orch--arch.md)
  - update: `maintainLeadership` goroutine description — check interval changed from `TTL/2` to `LeadershipDurationSeconds / 4`, add CAS retry logic (up to 3 attempts) with `retryIntervalOnCASErr = LeadershipDurationSeconds / 20`
  - update: Goroutine launch flow — `scheduleKiller()` is called proactively on acquisition and on each successful CAS renewal with `killTime = leadershipStartTime + LeadershipDurationSeconds * 0.8`, not reactively from `leadershipMonitor`
  - update: `scheduleKiller()` semantics — schedule only, no unschedule/disarm/discharge; killer must never be stopped because goroutines can continue working after VM context is cancelled
  - update: Key constants — add `maintainLeadershipCheckInterval`, `retryIntervalOnCASErr`, `maxCASRetries`, update `processKillThreshold` to `0.8 * LeadershipDurationSeconds`; update timing relationships diagram
  - update: Leadership acquisition flow — `insertIfNotExist()` followed by `scheduleKiller(leadershipStartTime + LeadershipDurationSeconds * 0.8)`
  - update: Leadership maintenance flow — on CAS error retry up to 2 more times; on success `scheduleKiller()`; on `!ok` `releaseLeadership()`

## Construction

- [x] update: [pkg/ielections/interface.go](../../../../../pkg/ielections/interface.go)
  - update: `AcquireLeadership` signature — remove `*KillerScheduler` parameter, killer is created internally

- [x] update: [pkg/ielections/types.go](../../../../../pkg/ielections/types.go)
  - add: unexported `killerScheduler` struct with fields: `ctx context.Context`, `cancel context.CancelFunc`, `clock timeu.ITime`

- [x] new: [pkg/ielections/impl_killer.go](../../../../../pkg/ielections/impl_killer.go)
  - add: `newKillerScheduler(clock timeu.ITime) *killerScheduler` — uses isolated clock in tests to avoid os.Exit on MockedTime advance
  - add: `scheduleKiller(deadline time.Time)` method — cancels previous killer, spawns goroutine that calls `os.Exit(1)` on timer or returns on context done

- [x] new: [pkg/ielections/consts.go](../../../../../pkg/ielections/consts.go)
  - add: `maxRetriesOnCASErr = 2`, `maintainIntervalDivisor = 4`, `retryIntervalDivisor = 20`, `preCASKillTimeFactor = 0.75`, `killDeadlineFactor = 0.8`
  - add: comments describing each constant

- [x] update: [pkg/ielections/impl.go](../../../../../pkg/ielections/impl.go)
  - update: `AcquireLeadership` — capture `leadershipStartTime := e.clock.Now()` before `InsertIfNotExist`; create `killerScheduler` internally via `newKillerScheduler(e.clock)`; schedule killer at `leadershipStartTime + LeadershipDurationSeconds * killDeadlineFactor` as `time.Time` deadline; pass killer to `maintainLeadership`
  - update: `maintainLeadership` — accept `*killerScheduler`; change check interval from `TTL/2` to `LeadershipDurationSeconds / maintainIntervalDivisor`; schedule pre-CAS killer with `preCASKillTimeFactor`; on CAS error retry up to `maxRetriesOnCASErr` more times with `retryIntervalOnCASErr = LeadershipDurationSeconds / retryIntervalDivisor`; on CAS success schedule killer with `killDeadlineFactor`; on `!ok` call `releaseLeadership()`
  - add: `durationMult` helper — converts seconds to nanoseconds before multiplying by factor to avoid float-to-int truncation

- [x] update: [pkg/ielections/provide.go](../../../../../pkg/ielections/provide.go)
  - update: No changes needed

- [x] update: [pkg/ielections/impl_testsuite.go](../../../../../pkg/ielections/impl_testsuite.go)
  - update: Adapt test suite to new `AcquireLeadership` signature — no longer pass `*killerScheduler`
  - update: Adjust timing in existing tests for new check interval (`LeadershipDurationSeconds / 4` instead of `TTL/2`)

- [x] update: [pkg/goutils/testingu/mocktime.go](../../../../../pkg/goutils/testingu/mocktime.go)
  - add: `NewIsolatedTime() timeu.ITime` method to `IMockTime` interface
  - add: `NewIsolatedTime()` implementation on `mockedTime` — returns a new independent `IMockTime` instance

- [x] update: [pkg/goutils/testingu/mocktime.md](../../../../../pkg/goutils/testingu/mocktime.md)
  - add: Isolated time design note

- [x] update: [pkg/vvm/impl_orch.go](../../../../../pkg/vvm/impl_orch.go)
  - update: `tryToAcquireLeadership` — call `elections.AcquireLeadership` without killer (created internally by elections)
  - remove: `killerRoutine` method — replaced by internal `killerScheduler.scheduleKiller`
  - update: `leadershipMonitor` — remove reactive `killerRoutine` spawning on leadership loss (killer is now scheduled proactively from `maintainLeadership`)

- [x] new: [pkg/ielections/impl_killer_test.go](../../../../../pkg/ielections/impl_killer_test.go)
  - add: `Test_newKillerScheduler` — verifies isolated clock
  - add: `Test_scheduleKiller` — verifies past/zero deadline rejection and previous killer cancellation
  - add: `Test_killerRescheduleOnCAS` — verifies killer rescheduling on CAS success, no rescheduling on CAS `!ok`, no rescheduling on CAS error after all retries

- [x] update: [pkg/ielections/impl_test.go](../../../../../pkg/ielections/impl_test.go)
  - add: `TestDurationMult` — verifies `durationMult(10, 0.75)` produces 7.5s
