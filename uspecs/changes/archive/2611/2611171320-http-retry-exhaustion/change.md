---
change_id: 2611171243-http-retry-exhaustion
type: feat
issue_url: https://untill.atlassian.net/browse/AIR-5284
domains: [prod]
scope: [apps]
---

# Change request: HTTP retry exhaustion reporting

Refs:

- [AIR-5284: voedger, httpu: expose HTTP retry exhaustion through](./issue-AIR-5284.md)

## Why

The Voedger application platform currently returns the last HTTP failure when the configured retry duration expires, without identifying retry exhaustion. Callers need a reliable signal to decide whether to retry at their own level without duplicating HTTP status classification.

## What

- HTTP callers can identify failures returned because the configured retry duration was exceeded while the failure remained retryable
- Federation callers retain the retry exhaustion signal after response conversion, including responses with empty or malformed bodies
- Returned failures preserve the underlying error information alongside the retry exhaustion signal
- Successful and caller-accepted responses, non-retryable failures, and context cancellation or deadline termination remain distinct from retry exhaustion
- Callers can detect retry exhaustion through standard error matching even after errors are wrapped

## How

Decisions:

- Keep retry-duration enforcement in the HTTP client layer; the generic retrier continues to manage attempts, backoff, and cancellation without owning a retry budget
- Extend the existing error contract without changing HTTP client or federation method signatures, adding response fields, or introducing a public retryability interface
- Preserve the current duration semantics: status-specific limits are measured from request start and checked after a response, rather than introducing a strict deadline that interrupts an in-flight request
- Carry exhaustion through the error chain and combine it with the converted federation failure, retaining typed server errors when decoding succeeds and conversion errors when decoding fails
- Validate the contract through the production HTTP client and federation against a local HTTP server, using configurable short retry windows rather than waiting for the default duration

Assumptions:

- None

Out of scope:

- Changing Workspace creation projectors or actualizer retry behavior
- Adding maximum durations or attempt limits to generic retries or transport-error retry policies
- Changing default retryable statuses, retry windows, backoff, or Retry-After handling

References:

- [HTTP retry timing and response processing](../../../../../pkg/goutils/httpu/impl.go)
- [Configurable status retry durations](../../../../../pkg/goutils/httpu/impl_opts.go)
- [Generic retry and cancellation behavior](../../../../../pkg/goutils/retry/impl.go)
- [Federation error conversion boundary](../../../../../pkg/coreutils/federation/utils.go)
- [Existing federation interface contract](../../../../../pkg/coreutils/federation/interface.go)
- [HTTP integration test infrastructure](../../../../../pkg/goutils/httpu/impl_test.go)
- [Federation integration test infrastructure](../../../../../pkg/coreutils/federation/impl_test.go)

## Construction

- [x] update: [goutils/httpu/impl_test.go](../../../../../pkg/goutils/httpu/impl_test.go)
  - Add regression coverage for configured status retry exhaustion through the production client, asserting both exhaustion and the underlying unexpected-status error remain detectable after wrapping
  - Cover successful retries, caller-accepted statuses, non-retryable terminal responses after retries, disabled retries, and context cancellation or deadline termination without the exhaustion marker
  - Cover exhaustion followed by a response-body read failure, preserving both the marker and the read error
  - Use short configurable retry durations and controlled server responses, avoiding assertions dependent on an exact jitter delay or attempt count

- [x] update: [coreutils/federation/impl_test.go](../../../../../pkg/coreutils/federation/impl_test.go)
  - Exercise retry-enabled federation calls through the real HTTP client against a local server until the configured duration expires
  - Verify exhaustion survives conversion for structured server errors, empty bodies, and malformed response bodies, including discarded-response requests
  - Assert decoded server errors and decoding errors remain inspectable alongside exhaustion
  - Verify success, expected conflict responses, and non-retryable failures retain their existing behavior without exhaustion

- [x] update: [goutils/httpu/errors.go](../../../../../pkg/goutils/httpu/errors.go)
  - Add the exported ErrRetryExhausted sentinel documenting that the active status retry policy exhausted its configured duration

- [x] update: [goutils/httpu/impl.go](../../../../../pkg/goutils/httpu/impl.go)
  - Capture exhaustion at the existing status-policy duration check without changing retry timing or configured policies
  - Wrap terminal failures with the sentinel while preserving the original error, including failures reading the final unexpected response
  - Preserve successful and caller-accepted responses and distinguish context termination from duration exhaustion

- [x] update: [coreutils/federation/utils.go](../../../../../pkg/coreutils/federation/utils.go)
  - Preserve the exhaustion marker across response conversion and all failing conversion exits without bypassing server-error decoding
  - Retain existing response acceptance behavior and error causes without adding status classification to federation

- [x] update: [coreutils/federation/interface.go](../../../../../pkg/coreutils/federation/interface.go)
  - Document exhaustion detection and distinguish it from context termination and permanent business rejection

- [x] update: [goutils/httpu/README.md](../../../../../pkg/goutils/httpu/README.md)
  - Document errors.Is detection and the caller's responsibility for any higher-level retry
  - Explain that absence of the marker does not establish that a failure is permanent

- [x] Verify: Run go test ./pkg/goutils/httpu ./pkg/coreutils/federation and review the cumulative changes for preserved error chains and unchanged retry policies

## Quick start

After a retry-enabled federation call, detect exhaustion using standard error matching:

```go
_, err := fed.Func(relativeURL, body)
if errors.Is(err, httpu.ErrRetryExhausted) {
    return err
}
```

The caller can propagate this error to an existing retrying workflow or schedule another attempt itself. This check does not start another retry and does not classify other errors as permanent.
