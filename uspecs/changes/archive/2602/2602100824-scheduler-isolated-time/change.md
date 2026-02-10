---
registered_at: 2026-02-09T15:05:44Z
change_id: 2602091505-scheduler-isolated-time
baseline: 4b8d03a79c656f7957444c04cebfd1a0c1a92c20
archived_at: 2026-02-10T08:24:51Z
---

# Change request: Isolate job scheduler time from global MockTime in tests

## Why

Jobs fire on every integration test run because the global MockTime is advanced by 24h on each next integration VIT test. This causes unintended job executions during tests, making test behavior unpredictable and harder to reason about.

## What

Decouple job scheduler timing from the global MockTime:

- When schedulers for jobs start, they use `ITime.NewIsolatedTime` and use it to schedule jobs
- Jobs will not fire automatically since the isolated time is not advanced by global MockTime changes

VIT gains explicit control over job scheduling:

- VIT provides a special function to advance the scheduler's isolated time so that jobs start on demand
