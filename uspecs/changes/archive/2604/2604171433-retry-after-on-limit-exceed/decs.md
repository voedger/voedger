# Decisions: Send Retry-After header on function limit exceed

## Retry-After value computation

Compute Retry-After from the limit's declared rate parameters in `IAppDef`, the same way `vit.RatePerPeriod()` does: look up the exceeded limit by the QName returned from `IsLimitExceeded`, read `Rate().Count()` and `Rate().Period()`, and use `Period / Count` as the per-token interval (confidence: high).

Rationale: This keeps all existing interfaces (`IBuckets`, `Limiter`, `IAppPartition.IsLimitExceeded`) untouched. The exceeded limit's QName is already returned by `IsLimitExceeded`, and `appdef.Limit(appDef.Type, limitName).Rate()` provides `Count()` and `Period()` that fully describe the refill schedule declared in VSQL. The bucket's internal token-refill state is an implementation detail of `iratesce`; the VSQL-declared rate is the contractual throughput the client can rely on.

Alternatives:

- Dynamic value from bucket's next-token time via extended limiter API (confidence: medium)
  - More precise but requires changing `IBuckets.TakeTokens`, `Limiter.Exceeded`, and `IAppPartition.IsLimitExceeded` signatures and all their callers and tests
- Fixed default (e.g., 1 second) like existing 503 path (confidence: low)
  - Simplest, but gives clients no real information and can cause retry storms

## Propagation of Retry-After from processor to HTTP response

Extend `coreutils.SysError` with an optional `Headers map[string]string` field and an `AddHeader(...)` method; the processor attaches `Retry-After` to the 429 `SysError`, and the router copies these headers onto the HTTP response when rendering the error (confidence: high).

Rationale: The `SysError` is already the single value that flows from processor to router as the error payload (see `bus.ReplyErrDef` → `responder.Respond` → `reply_v1`/`reply_v2`). Attaching transport-level hints to it avoids adding new fields to `bus.ResponseMeta` or special casing by status code in the router. `AddHeader` keeps the call sites concise and the field optional, so existing `SysError` usages are unaffected.

Alternatives:

- New optional `RetryAfter time.Duration` / `Headers` field in `bus.ResponseMeta` rendered by `initResponse` (confidence: medium)
  - Keeps `SysError` a pure domain error but requires threading the value through `Responder.Respond` call sites and router's `initResponse`
- Hard-code the Retry-After in the router by inspecting status 429 and calling back into `appPart`/`IAppDef` (confidence: low)
  - Breaks layering: router should not depend on rate-limit semantics

## Scope of processors affected

Apply the change to the three existing `IsLimitExceeded` call sites that return 429: command processor (`pkg/processors/command/impl.go#limitCallRate`), query processor v1 (`pkg/processors/query/impl.go`), query processor v2 (`pkg/processors/query2/util.go#queryRateLimitExceeded`) (confidence: high).

Rationale: These are all the code paths the issue refers to ("processors: limit exceeded"). The 503 "no query processors available" path in `pkg/vvm/impl_requesthandler.go` and the router-level QP wsQueryLimiter `replyServiceUnavailable` already send `Retry-After` and are out of scope of AIR-3603.

Alternatives:

- Also unify the 503 Retry-After path to use the same mechanism (confidence: medium)
  - Valuable cleanup but expands scope beyond the issue
- Limit to a single processor first (e.g., query v2) and replicate later (confidence: low)
  - Leaves behavior inconsistent across processors for an extended period

## Header format and minimum value

Send `Retry-After` as an integer number of seconds (delta-seconds form), rounded up (ceiling) from the computed duration, with a minimum of 1 (confidence: high).

Rationale: RFC 9110 allows both delta-seconds and HTTP-date; delta-seconds is simpler to produce, easier to test deterministically, and matches both the existing `replyServiceUnavailable` implementation and `parseRetryAfterHeader` test expectations in `pkg/goutils/httpu`. Ceiling avoids 0-second values that would encourage immediate retries; a 1-second floor guarantees a real back-off.

Alternatives:

- HTTP-date format (confidence: low)
  - Harder to test (depends on wall clock), more parsing surface on clients
- Floor with 1-second minimum (confidence: medium)
  - Close to equivalent; slightly under-estimates wait and can trigger an extra retry
