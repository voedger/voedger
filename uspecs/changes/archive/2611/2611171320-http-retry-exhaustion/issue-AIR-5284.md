# voedger, httpu: expose HTTP retry exhaustion through

- URL: https://untill.atlassian.net/browse/AIR-5284
- ID: AIR-5284
- State: In progress
- Author: Denis Gribanov
- Labels: none
- Assignee: Denis Gribanov
- Parent issue: [AIR-4959: voedger: partition recovery performance](https://untill.atlassian.net/browse/AIR-4959)

## Description

**Why**

When the maximum retry duration expires, `httpu` returns the last failure without indicating that retries were exhausted. Callers cannot distinguish this outcome from other failures without inspecting HTTP status codes and duplicating retry-policy logic.

**What**

* Introduce `httpu.ErrRetryExhausted`, detectable through `errors.Is`
* Attach the sentinel when a retryable failure is returned because its configured maximum retry duration was exceeded
* Preserve the underlying error and the exhaustion marker through federation response conversion, including empty or malformed response bodies
* Keep success, non-retryable failures, and context cancellation distinct from retry exhaustion
* Add tests covering exhaustion detection and propagation through `IFederationWithRetry.Func()`
