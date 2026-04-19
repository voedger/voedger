# Logger

Structured and legacy logging with hierarchical level filtering,
automatic caller tracking, and context-aware attribute propagation.

## Problem

Go's standard `log` package lacks level-based filtering, caller
tracking, and context attribute propagation, forcing developers
to implement repetitive logging infrastructure manually.

<details>
<summary>Without logger</summary>

```go
// Manual level checking and formatting - boilerplate everywhere
var logLevel int = 3 // Info level
var mu sync.Mutex

func logError(args ...interface{}) {
    if logLevel >= 1 { // boilerplate: manual level check
        mu.Lock() // boilerplate: manual thread safety
        defer mu.Unlock()
        // boilerplate: manual stack trace
        pc, _, line, _ := runtime.Caller(1)
        fn := runtime.FuncForPC(pc)
        funcName := "unknown"
        if fn != nil {
            funcName = fn.Name()
        }
        timestamp := time.Now().Format("01/02 15:04:05.000")
        fmt.Fprintf(os.Stderr, "%s: ERROR: [%s:%d]: %s\n",
            timestamp, funcName, line, fmt.Sprint(args...))
    }
}

// boilerplate: duplicate code for every level
func logInfo(args ...interface{}) { /* same boilerplate */ }

// Structured attributes must be threaded manually through calls
func handleRequest(reqID int64, wsID int64) {
    // boilerplate: pass ids explicitly to every log call
    logInfo("started requestID=%d wsID=%d", reqID, wsID)
    processPayment(reqID, wsID) // must thread ids everywhere
}

func processPayment(reqID int64, wsID int64) {
    // boilerplate: repeat ids in every log call
    logInfo("payment processed reqID=%d wsID=%d", reqID, wsID)
}
```

</details>

<details>
<summary>With logger</summary>

```go
import (
    "context"
    "github.com/voedger/voedger/pkg/goutils/logger"
)

func handleRequest(ctx context.Context, reqID, wsID int64) {
    ctx = logger.WithContextAttrs(ctx, map[string]any{
        logger.LogAttr_ReqID: reqID,
        logger.LogAttr_WSID:  wsID,
    })
    logger.InfoCtx(ctx, "started")  // attrs included automatically
    processPayment(ctx)             // just pass ctx
}

func processPayment(ctx context.Context) {
    logger.InfoCtx(ctx, "payment processed") // attrs still there
}
```

</details>

Consumers of stdlib `*log.Logger` (e.g. `http.Server.ErrorLog`,
`acme/autocert`, third-party libraries) have no first-class way to
forward byte-oriented output into the structured logger, so teams
reinvent ad-hoc `io.Writer` adapters that lose ctx attrs, misreport
the caller frame, and split multi-line panic traces into
disconnected records.

<details>
<summary>Without NewStdErrorLogBridge</summary>

```go
// Ad-hoc adapter: no ctx attrs, wrong src frame, no stage,
// per-line splitting fragments net/http panic stack traces.
type filteringWriter struct {
    out     io.Writer
    filters []string
}

func (f *filteringWriter) Write(p []byte) (int, error) {
    for _, s := range f.filters { // boilerplate: substring drop
        if bytes.Contains(p, []byte(s)) {
            return len(p), nil
        }
    }
    trimmed := bytes.TrimRight(p, "\r\n") // boilerplate: CRLF trim
    if len(trimmed) == 0 {
        return len(p), nil
    }
    // boilerplate: splits multi-line payloads, each line logged
    // separately so a single panic stack becomes N disconnected
    // records; src attribute points at this Write, not the caller
    for _, line := range bytes.Split(trimmed, []byte("\n")) {
        fmt.Fprintln(f.out, string(line))
    }
    return len(p), nil
}

srv := &http.Server{
    ErrorLog: log.New(&filteringWriter{
        out:     os.Stderr,
        filters: []string{"TLS handshake error"},
    }, "", 0),
}
```

</details>

<details>
<summary>With NewStdErrorLogBridge</summary>

```go
import (
    "context"
    "net/http"

    "github.com/voedger/voedger/pkg/goutils/logger"
)

func newServer(ctx context.Context) *http.Server {
    return &http.Server{
        ErrorLog: logger.NewStdErrorLogBridge(ctx, "http",
            logger.WithFilter([]string{"TLS handshake error"})),
    }
}
```

</details>

## Features

- **Level filtering** - Hierarchical levels (`Error`→`Trace`) with
  atomic switching and restore-on-defer
  - [Level constants: logger.go#L22](logger.go#L22)
  - [Atomic level check: impl.go#L37](impl.go#L37)
  - [Level restoration: logger.go#L36](logger.go#L36)
- **Caller tracking** - Automatic function name and line number
  in every log entry
  - [Stack frame analysis: impl.go#L42](impl.go#L42)
  - [Formatted output: impl.go#L55](impl.go#L55)
  - [Skip frame control: logger.go#L64](logger.go#L64)
- **Context-aware logging** - `*Ctx` functions read logging key/value
  pairs stored in `context.Context` and append them to each entry
  - [WithContextAttrs: loggerctx.go#L23](loggerctx.go#L23)
  - [Ctx logging functions: loggerctx.go#L31](loggerctx.go#L31)
  - [sLogAttrsFromCtx: loggerctx.go#L87](loggerctx.go#L87)
  - **Predefined attribute key constants** ([consts.go](consts.go#L19))

    | Constant            | Key         | Example value |
    |---------------------|-------------|---------------|
    | `LogAttr_VApp`      | `vapp`      | `my.app`      |
    | `LogAttr_Feat`      | `feat`      | `magicmenu`   |
    | `LogAttr_ReqID`     | `reqid`     | `42`          |
    | `LogAttr_WSID`      | `wsid`      | `1001`        |
    | `LogAttr_Extension` | `extension` | `myFunc`      |

- **Output customization** - Pluggable `PrintLine` with automatic
  stderr/stdout routing per level
  - [PrintLine hook: logger.go#L88](logger.go#L88)
  - [DefaultPrintLine: logger.go#L93](logger.go#L93)
- **[Performance guards](logger.go#L68)** - `IsVerbose()`,
  `IsTrace()`, etc. prevent computing expensive arguments
- **slog level mapping** - Both `Verbose` and `Trace` internal levels
  map to `slog.LevelDebug` when emitting structured log records
- **Log capturing** - in-memory buffer that intercepts log output
  during tests; auto-restored when the test ends
  - [StartCapture: logcapture.go#L13](logcapture.go#L13)
  - [ILogCaptor: types.go#L28](types.go#L28)
  - [HasLine: logcapture.go#L30](logcapture.go#L30)
  - [EventuallyHasLine: logcapture.go#L41](logcapture.go#L41)
  - [NotContains: logcapture.go#L78](logcapture.go#L78)
- **Stdlib `*log.Logger` bridge** - forward byte-oriented stdlib
  output (e.g. `http.Server.ErrorLog`) into the voedger ctx logger
  at Error level with one `Write` = one record, trailing CRLF trim,
  embedded newlines preserved, and optional substring filtering
  - [NewStdErrorLogBridge: stdlogbridge.go#L25](stdlogbridge.go#L25)
  - [WithFilter: stdlogbridge.go#L14](stdlogbridge.go#L14)
  - [stdLogBridgeWriter.Write: stdlogbridge.go#L33](stdlogbridge.go#L33)

## Use

See [legacy functions basic usage](logger_test.go#L31)
and [slog functions basic usage](logger_test.go#L43)

## Example output

```text
09/29 13:29:04.355: *****: [core-logger.TestLegacyFuncs_BasicUsage:22]: Error
09/29 13:29:04.374: ===: [core-logger.TestSlogFuncs_BasicUsage:24]: My info
time=2026-03-24T14:05:26.461+03:00 level=INFO msg="started" src=myapp.handleRequest:12 reqid=42 wsid=1001
```

## Links

- [Why does the TRACE level exist, and when should I use it rather than DEBUG?](https://softwareengineering.stackexchange.com/questions/279690/why-does-the-trace-level-exist-and-when-should-i-use-it-rather-than-debug)
  - [Good answer](https://softwareengineering.stackexchange.com/questions/279690/why-does-the-trace-level-exist-and-when-should-i-use-it-rather-than-debug/360810#360810)

