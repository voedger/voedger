# Decisions: Fix body loss on retry in HTTPClient

## Body buffering strategy

Buffer `bodyReader` into `[]byte` via `io.ReadAll` in `req()` before the retry loop, then create a fresh `bytes.NewReader` on each retry iteration (confidence: high).

Rationale: The retry closure in `req()` reuses `opts.bodyReader` across attempts. `io.Reader` is consumed after the first read. Buffering once and replaying via `bytes.NewReader` is the simplest fix that keeps the change localized to `req()` without touching `newRequest`, `ReqReader`, or the `IHTTPClient` interface.

Alternatives:

- Change `reqOpts.bodyReader` and `IHTTPClient.ReqReader` to accept `io.ReadSeeker`, seek to start before each retry (confidence: medium)
  - More memory-efficient for large bodies but breaks multiple callers (`impl_http_storage.go`, `federation/impl.go`, `impl_blob_test.go`) that pass plain `io.Reader`
- Use `io.TeeReader` to capture bytes during first read, replay from buffer on retries (confidence: low)
  - More complex with no benefit over upfront buffering, harder to reason about partial reads on transport errors

## Placement of buffering code within `req()`

Place buffering after `compileOpts` returns and before the retry loop (confidence: high).

Rationale: `opts.bodyReader` is only populated after `compileOpts` resolves all option functions. Buffering immediately after keeps I/O separate from option compilation and ensures the `[]byte` is ready before any retry code runs.

Alternatives:

- Buffer inside `ReqReader` before calling `req()` (confidence: medium)
  - `req()` still needs to know about buffered bytes to create fresh readers per attempt, splits the concern across two methods
- Buffer inside `compileOpts` (confidence: low)
  - Mixes I/O with option compilation, violates single responsibility

## Handling nil bodyReader

Skip buffering when `opts.bodyReader` is nil and let the existing `body string` path in `newRequest` handle it (confidence: high).

Rationale: `body` and `bodyReader` are mutually exclusive (documented in `newRequest`). When `bodyReader` is nil, `newRequest` already creates a fresh `bytes.NewReader([]byte(body))` on each call, which is inherently retry-safe. No buffering needed.

Alternatives:

- Always buffer both paths into `[]byte` (confidence: low)
  - Unnecessary duplication; the `string` body path already works correctly

## Memory impact of full body buffering

Accept full body buffering without size limits (confidence: high).

Rationale: Current callers pass small payloads — `impl_http_storage.go` passes `bytes.NewReader(kb.body)` (already a `[]byte`), blob tests pass small test data. The `Req(string)` method already holds the entire body in memory as a string. Buffering the reader is consistent.

Alternatives:

- Add a configurable max body size and return error if exceeded (confidence: medium)
  - Adds complexity with no current need; no caller sends large streaming bodies through `ReqReader` with retry enabled
- Document that retries are unsupported with `ReqReader` (confidence: low)
  - Leaves the bug in place; retry policies apply by default, violating principle of least surprise

## Test approach for reproducing the bug

Use a test server that returns 503 on first attempt and 200 on second, verifying the request body is identical on both attempts (confidence: high).

Rationale: This directly exercises the retry-on-status path with `ReqReader`, which is the exact scenario described in the issue. Comparing captured bodies from both attempts proves whether the body survives the retry.

Alternatives:

- Test via error-based retry (`retryOnErr`) instead of status-based retry (confidence: medium)
  - Also valid but status-based retry is the more common real-world scenario and exercises the `discardRespBody` + `errRetry` path
- Mock `http.Client` instead of using a real HTTP server (confidence: low)
  - Loses end-to-end coverage of the actual `newRequest` → `Do` → retry flow
