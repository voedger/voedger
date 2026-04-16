# Implementation plan: Fix superfluous response.WriteHeader in n10n handler

## Construction

- [x] update: [impl_n10n.go](../../../../../pkg/router/impl_n10n.go)
  - fix: Replace `WriteTextResponse` with `writeResponse` using SSE event format in subscribe error path
  - add: Comment clarifying 200 OK header is committed by `rw.Write()`
- [x] update: [impl_test.go](../../../../../pkg/router/impl_test.go)
  - add: `TestSubscribeAndWatch_NoSuperfluousWriteHeader` using real `in10nmem.NewN10nBroker` with zero subscription quota
  - add: `httptest.NewUnstartedServer` with HTTP error log redirected to `logger.StartCapture`
  - add: Assert captured logs do not contain "superfluous response.WriteHeader"
- [x] update: [types.go](../../../../../pkg/goutils/logger/types.go)
  - add: `io.Writer` to `ILogCaptor` interface
