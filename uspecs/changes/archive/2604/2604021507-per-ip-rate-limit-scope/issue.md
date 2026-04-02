# AIR-3420: Use PER IP rate limit scope

**Source:** [AIR-3420](https://untill.atlassian.net/browse/AIR-3420)

## Description

There are two related problems with IP-scoped rate limiting:

### 1. `ResetLimits` ignores `RateScope_IP`

The `ResetLimits` method in `pkg/appparts/internal/limiter/limiter.go` does not use the `RateScope_IP` scope when constructing bucket keys. The `Exceeded` method correctly sets `key.RemoteAddr` when the rate has `RateScope_IP`, but `ResetLimits` omits this, leading to a mismatch in how rate limit buckets are identified and reset.

This means when a rate limit with IP scope is reset, the reset targets a bucket key without the `RemoteAddr` field, which does not match the bucket key used during token consumption. As a result, IP-scoped rate limits are not effectively reset.

### 2. `IsLimitExceeded` callers pass empty `remoteAddr`

Both the command processor (`limitCallRate` in `pkg/processors/command/impl.go`) and the query processor (`check function call rate` in `pkg/processors/query/impl.go`) call `IsLimitExceeded` with an empty string `""` as `remoteAddr`. This means IP-scoped rate limits are never properly enforced — all requests share the same empty-address bucket instead of being tracked per IP.

### Investigation needed

It is necessary to investigate where the actual `remoteAddr` value should be sourced from in each call site (e.g., from the request message, HTTP headers, bus message metadata, etc.).

