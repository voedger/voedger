---
registered_at: 2026-02-07T12:41:12Z
change_id: 2602071540-leadership-retry-recovery
baseline: 47357a838e24dcf052ea5bbcd4210c0db87da6fb
---

# Change request: Improve maintainLeadership with retry and scheduled killer

## Why

Even a short-term network problem causes the server to reboot because `maintainLeadership` releases leadership immediately on a single `CompareAndSwap` failure. The server should wait some time for possible recovery instead of failing fast, and the killer must be scheduled proactively rather than reactively.

## What

Redesign the `maintainLeadership` goroutine to be resilient to transient failures:

- Increase check frequency: `maintainLeadershipCheckInterval = LeadershipDurationSeconds / 4`
- Add CAS retry interval: `retryIntervalOnCASErr = LeadershipDurationSeconds / 20`
- Schedule killer proactively before each `CompareAndSwap` call with `killTime = leadershipStartTime + LeadershipDurationSeconds * 0.8`
- On CAS error, retry up to 2 more times (3 attempts total) before releasing leadership
- `scheduleKiller()` only — no unschedule/disarm/discharge; killer must never be stopped because some goroutines can continue working even after VM context is cancelled

On leadership acquisition for the first time:

- `leadershipStartTime := now()`
- `insertIfNotExist()`
- `killTime = leadershipStartTime + LeadershipDurationSeconds * 0.8`
- `scheduleKiller(killTime)`

On leadership maintenance:

- `leadershipStartTime := now()`
- `ok, err := compareAndSwap()`
- If error: retry `compareAndSwap()` up to 2 more times
- If ok: `killTime = leadershipStartTime + LeadershipDurationSeconds * 0.8` then `scheduleKiller(killTime)`
- If not ok: `releaseLeadership()`

Update `vvm-orch--arch.md` accordingly.
