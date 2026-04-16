---
registered_at: 2026-04-15T19:45:12Z
change_id: 2604151945-fix-superfluous-writeheader
baseline: 7469ac179a631a24474dafd5a5077121787d9b47
issue_url: https://untill.atlassian.net/browse/AIR-3528
archived_at: 2026-04-16T09:24:40Z
---


# Change request: Fix superfluous response.WriteHeader in n10n handler

## Why

The `subscribeAndWatchHandler` in the router writes the channel ID to the response using `fmt.Fprintf(rw, ...)`, which implicitly commits a 200 OK status header via `rw.Write()`. If a subsequent `Subscribe` call fails, the error path calls `WriteTextResponse`, which attempts to call `rw.WriteHeader(code)` again. This produces the runtime warning:

```
http: superfluous response.WriteHeader call from github.com/voedger/voedger/pkg/router.WriteTextResponse (utils.go:36)
```

The HTTP status code from the error is silently discarded because headers are already committed, so the client always sees a 200 status regardless of the subscribe failure.

See [issue.md](issue.md) for details.

## What

Changes in `pkg/router/impl_n10n.go`:

- In the `subscribeAndWatchHandler` subscribe loop error path, replace `WriteTextResponse` (which calls `w.WriteHeader`) with `writeResponse` using SSE event format, since the response is already in SSE streaming mode at that point

Changes in `pkg/router/impl_test.go`:

- Add `TestSubscribeAndWatch_NoSuperfluousWriteHeader` that uses a real `in10nmem.NewN10nBroker` with zero subscription quota, an `httptest` server with HTTP error log redirected to `logger.StartCapture`, and asserts captured logs do not contain the superfluous WriteHeader message

Changes in `pkg/goutils/logger/types.go`:

- Add `io.Writer` to `ILogCaptor` interface to allow using it as `log.New()` output
