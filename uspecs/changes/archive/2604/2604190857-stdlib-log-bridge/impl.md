# Implementation plan: logger bridge between goutils/logger and stdlib log.Logger

## Construction

- [x] update: [pkg/goutils/logger/consts.go](../../../../../pkg/goutils/logger/consts.go)
  - add: unexported constant `stdLogBridgeSkipStackFrames = 3`
- [x] create: [pkg/goutils/logger/stdlogbridge.go](../../../../../pkg/goutils/logger/stdlogbridge.go)
  - add: `NewStdErrorLogBridge(ctx, stage, opts...) *log.Logger` that constructs the writer with `logLevel: LogLevelError`
  - add: `Write` method that applies `WithFilter` substrings, trims trailing `\r`/`\n`, drops empty payloads, forwards via `LogCtx(ctx, stdLogBridgeSkipStackFrames, w.logLevel, stage, payload)` exactly once per call
  - add: `WithFilter(substrings []string) StdLogBridgeOption`
- [x] update: [pkg/goutils/logger/types.go](../../../../../pkg/goutils/logger/types.go)
  - add: unexported `stdLogBridgeWriter` struct with `ctx`, `stage`, `logLevel`, `filters` fields
  - add: exported `StdLogBridgeOption func(*stdLogBridgeWriter)`
- [x] update: [pkg/goutils/logger/logger_test.go](../../../../../pkg/goutils/logger/logger_test.go)
  - add: `Test_NewStdLogBridge` with subtests for single-line forwarding (stage + ctx attrs + src), multi-line preservation (`msg="first\nsecond"`, CRLF trimmed, one line), empty-after-trim suppression, disabled level suppression, `WithFilter` drop behavior
- [x] update: [pkg/goutils/logger/README.md](../../../../../pkg/goutils/logger/README.md)
  - add: feature bullet under `## Features` linking into `stdlogbridge.go`
  - add: before/after `<details>` pair under `## Problem` — DIY `io.Writer` adapter versus `NewStdErrorLogBridge(..., WithFilter(...))`
- [x] Review

## Quick start

Forward `http.Server` stdlib error logging into the voedger structured logger, dropping noisy TLS handshake lines:

```go
import (
    "context"
    "net/http"

    "github.com/voedger/voedger/pkg/goutils/logger"
)

ctx := logger.WithContextAttrs(context.Background(), map[string]any{
    logger.LogAttr_VApp: "my.app",
})

srv := &http.Server{
    Addr: ":8080",
    ErrorLog: logger.NewStdErrorLogBridge(ctx, "http",
        logger.WithFilter([]string{"TLS handshake error"})),
}
```
