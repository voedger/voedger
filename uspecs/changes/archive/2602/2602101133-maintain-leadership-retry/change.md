---
registered_at: 2026-02-10T08:50:58Z
change_id: 2602100850-maintain-leadership-retry
baseline: 324d14190b40d6d81d34cc5d2b19f3c7d642bf83
archived_at: 2026-02-10T11:33:57Z
---

# Change request: maintainLeadership should check frequently and wait for recovery

## Why

Currently `maintainLeadership` renews leadership every `TTL/2` seconds. If a single `CompareAndSwap` call fails (e.g. transient network issue), leadership is immediately released. With a 20s TTL the renewal happens every 10s, leaving no room for recovery. The renewal interval should be shortened to `TTL/4` and within each interval the system should retry `CompareAndSwap` every second, giving multiple chances to recover before giving up.

## What

Changes to `maintainLeadership` in `pkg/ielections/impl.go`:

- Change `tickerInterval` from `TTL/2` to `TTL/4`
- On each tick, instead of a single `CompareAndSwap` attempt, retry every second during the `tickerInterval` window
- Retry on error only, fail fast on !ok
- Check context on each retry iteration
- Release leadership only if all retry attempts within the interval fail

Tests in `pkg/ielections/impl_testsuite.go`:

- Add test for transient `CompareAndSwap` failure followed by recovery (leadership retained)
- Update existing tests to account for the new `TTL/4` interval and retry behavior
