# AIR-3528: Fix "http: superfluous response.WriteHeader" error

- **Type:** Task
- **Status:** In Progress
- **Assignee:** d.gribanov@dev.untill.com

## Description

The `subscribeAndWatchHandler` in the router writes the channel ID to the response using `fmt.Fprintf(rw, ...)`, which implicitly commits a 200 OK status header via `rw.Write()`. If a subsequent `Subscribe` call fails, the error path calls `WriteTextResponse`, which attempts to call `rw.WriteHeader(code)` again. This produces the runtime warning:

```
http: superfluous response.WriteHeader call from github.com/voedger/voedger/pkg/router.WriteTextResponse (utils.go:36)
```

The HTTP status code from the error is silently discarded because headers are already committed, so the client always sees a 200 status regardless of the subscribe failure.

## Resolution

- In `pkg/router/impl_n10n.go`, replace `WriteTextResponse` with `writeResponse` using SSE event format in the subscribe error path
- Add `TestSubscribeAndWatch_NoSuperfluousWriteHeader` to `pkg/router/impl_test.go` using real broker with zero subscription quota and `logger.StartCapture` to assert no superfluous WriteHeader in HTTP server logs
- Add `io.Writer` to `ILogCaptor` interface in `pkg/goutils/logger/types.go`
