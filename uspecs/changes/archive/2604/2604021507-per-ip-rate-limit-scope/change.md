---
registered_at: 2026-03-27T15:41:48Z
change_id: 2603271541-per-ip-rate-limit-scope
baseline: ebd38ec250cc672fa7f06591c39c47b859e00d2c
issue_url: https://untill.atlassian.net/browse/AIR-3420
archived_at: 2026-04-02T15:07:31Z
---

# Change request: Use PER IP rate limit scope

## Why

The limiter's `ResetLimits` method does not account for `RateScope_IP` when building bucket keys, while the `Exceeded` method does. This inconsistency means that IP-scoped rate limits are never properly reset, causing incorrect rate limiting behavior after a reset is requested. The same problem exists with `IAppPartition.IsLimitExceeded` — both command and query processors pass an empty string as `remoteAddr`, so IP-scoped rate limits are never properly enforced.

See [issue.md](issue.md) for details.

## What

Fix IP-scoped rate limit handling across the codebase:

- Add `remoteAddr` parameter to `Limiter.ResetLimits` and `IAppPartition.ResetRateLimit` so they can build IP-scoped bucket keys
- Set `key.RemoteAddr` when the limit rate has `RateScope_IP` in `ResetLimits`, matching the logic already present in `Exceeded`
- Pass actual `remoteAddr` values (instead of empty strings) in command processor (`limitCallRate`) and query processor (`check function call rate`) when calling `IsLimitExceeded`
- Update all callers of `ResetRateLimit` to pass the remote address
- Source `remoteAddr` from `http.Request.RemoteAddr` at the router level, stripping the port via `net.SplitHostPort` (rate limiter needs IP only)
