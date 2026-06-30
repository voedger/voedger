---
change_id: 2606291233-correct-retry-after-header-order
type: fix
issue_url: https://untill.atlassian.net/browse/AIR-4175
domains: [prod]
scope: [routing]
---

# Change request: Correct Retry-After header emission on router error responses

Refs:

- [AIR-4175: correct order for Retry-After header](./issue-AIR-4175.md)

## Why

The BLOB handler emits the `Retry-After` header after writing the status code, so the header is silently dropped and clients retrying a busy BLOB endpoint never learn how long to wait. The same root concern shows up in other router error paths: a raw `"Retry-After"` string literal instead of the `httpu.RetryAfter` constant, and error writers that render a `SysError` without applying its headers — so a processor-provided `Retry-After` (on a 429 rate-limit error) is also dropped. All of these should be made consistent so every back-off-eligible response carries `Retry-After`.

## What

Symptom: A client that receives a back-off-eligible error from the router — `503 Service Unavailable` from a busy BLOB endpoint, or a `429 Too Many Requests` rate-limit error — gets no `Retry-After` header, so it cannot back off for the intended interval.

```text
client sends request to the router
      |
      +--> BLOB read/write endpoint reports service unavailable
      |          |
      |          v
      |    blob handler: rw.WriteHeader(503) then rw.Header().Add("Retry-After", ...)
      |          <-- fault A: header is added after the status line is written, so net/http discards it
      |
      +--> a processor returns a 429 SysError carrying Retry-After (httpu.RetryAfter)
                 |
                 v
           error rendered via writeCommonError_V1 (BLOB and HTTP paths)
                 <-- fault B: SysError headers are never applied before WriteHeader, so Retry-After is dropped
                 |
                 v
client receives the error without Retry-After   (symptom)
```

Corrected behavior: Every router path that emits a back-off-eligible error sets `Retry-After` (and any other headers) before writing the status code, referencing the `httpu.RetryAfter` constant instead of a raw string. The `writeCommonError_V1` writer (V1 BLOB / HTTP error path) and the `replyErr` writer (V2 BLOB error path) apply the `SysError` headers before `WriteHeader`, so a processor-provided `Retry-After` reaches the client on BLOB and HTTP error responses as well.

## How

Decisions:

- Fix the ordering at the emission site in the BLOB handlers by reusing the already-correct `replyServiceUnavailable` helper instead of duplicating the header-then-status sequence inline
- Reference the `httpu.RetryAfter` constant everywhere a `Retry-After` header name is written, replacing the raw `"Retry-After"` literal (the `replyServiceUnavailable` helper)
- Reuse the existing `applySysErrorHeaders` helper to propagate `SysError` headers before the status line in `writeCommonError_V1` (V1 BLOB and HTTP error responders) and in `replyErr` (V2 BLOB error responder)
- Leave the processors unchanged — they already attach `Retry-After` to the `SysError` via `httpu.RetryAfter`

Out of scope:

- Changing the computed `Retry-After` durations or `DefaultRetryAfterSecondsOn503`
- Switching the header value between delta-seconds and HTTP-date form
- Any non-`Retry-After` header handling

References:

- [BLOB request handlers with the ordering fault](../../../../../pkg/router/impl_blob.go)
- [common error writers and replyServiceUnavailable](../../../../../pkg/router/utils.go)
- [applySysErrorHeaders helper](../../../../../pkg/router/impl_http.go)
- [httpu.RetryAfter constant](../../../../../pkg/goutils/httpu/consts.go)
- [router Retry-After response test](../../../../../pkg/router/impl_test.go)
- [integration test for rate-limit Retry-After](../../../../../pkg/sys/it/impl_rates_test.go)
- [RFC 9110 §10.2.3 Retry-After](https://datatracker.ietf.org/doc/html/rfc9110#section-10.2.3)

## Construction

- [x] update: [router/impl_test.go](../../../../../pkg/router/impl_test.go)
  - add: regression asserting BLOB write and read `503` responses carry the `Retry-After` header (fault A)
  - add: case asserting a `SysError` carrying `Retry-After` rendered through `writeCommonError_V1` (V1 BLOB / HTTP error path) propagates the header
  - add: case asserting a `SysError` carrying `Retry-After` rendered through `replyErr` (V2 BLOB error path) propagates the header

- [x] update: [sys/it/impl_rates_test.go](../../../../../pkg/sys/it/impl_rates_test.go)
  - switch the existing query `503` `Retry-After` assertion from the raw `"Retry-After"` literal to the `httpu.RetryAfter` constant, keeping the rest of the file consistent

- [x] update: [router/impl_blob.go](../../../../../pkg/router/impl_blob.go)
  - replace the inline `Retry-After`-then-`503` sequence in both handlers with a call to the `replyServiceUnavailable` helper, fixing the ordering and removing the duplication

- [x] update: [router/utils.go](../../../../../pkg/router/utils.go)
  - `replyServiceUnavailable`: use the `httpu.RetryAfter` constant instead of the `"Retry-After"` literal
  - `writeCommonError_V1`: apply `applySysErrorHeaders` for the unwrapped `SysError` before `WriteHeader`
  - `replyErr`: apply `applySysErrorHeaders` before delegating to `ReplyJSON`, so the V2 BLOB error path propagates `SysError` headers
