---
uspecs.registered_at: 2026-01-20T14:55:14Z
uspecs.change_id: 260120-duration-type-stores-days
uspecs.baseline: 86d0f721270a126701d4e348a965dd1f8d55216e
uspecs.archived_at: 2026-01-20T15:09:12Z
---

# Change request: Temporary BLOBs DurationType stores days directly

## Problem

DurationType currently stores the square root of days and uses the formula `dt*dt*secondsInDay` in its `Seconds()` method. This design is counterintuitive - when you want to represent N days, you need to pass sqrt(N) as the DurationType value.

## Solution overview

- Change DurationType to store the actual number of days directly. The `Seconds()` method should use the formula `dt*secondsInDay` instead of `dt*dt*secondsInDay`, making the type more intuitive to use.
- update existing integration tests in sys_it package
