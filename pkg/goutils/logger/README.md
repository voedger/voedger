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

<details>
<summary>Without logger (stdlib bridge)</summary>

```go
// Route http.Server.ErrorLog into a structured logger - DIY adapter
type ctxWriter struct{ ctx context.Context }

func (w *ctxWriter) Write(p []byte) (int, error) {
    rest := p
    for len(rest) > 0 {
        var line []byte
        if i := bytes.IndexByte(rest, '\n'); i >= 0 {
            line, rest = rest[:i], rest[i+1:]
        } else {
            line, rest = rest, nil
        }
        if len(line) > 0 && line[len(line)-1] == '\r' {
            line = line[:len(line)-1] // boilerplate: trim CRLF
        }
        // boilerplate: filter noisy lines manually
        if bytes.Contains(line, []byte("TLS handshake error")) {
            continue
        }
        if len(line) == 0 {
            continue
        }
        slog.ErrorContext(w.ctx, string(line)) // no stage, no src
    }
    return len(p), nil
}

// boilerplate: tune prefix/flags so slog doesn't duplicate timestamps
srv.ErrorLog = log.New(&ctxWriter{ctx: ctx}, "", 0)
```

</details>

<details>
<summary>With logger (stdlib bridge)</summary>

```go
import "github.com/voedger/voedger/pkg/goutils/logger"

srv.ErrorLog = logger.NewStdlibLogBridge(ctx, "endpoint.http.error",
    logger.WithFilter([]string{"TLS handshake error"}))
```

</details>

<details>
<summary>Without logger (log capturing in tests)</summary>

```go
// Intercept log output - manual snapshot AND restore of every hook
r, w, _ := os.Pipe()
oldStderr := os.Stderr
os.Stderr = w
oldLevel := logger.SetLogLevel(logger.LogLevelInfo)

var buf bytes.Buffer
done := make(chan struct{})
go func() {
    _, _ = io.Copy(&buf, r)
    close(done)
}()

// CRITICAL cleanup: miss a step and state leaks into the next test
t.Cleanup(func() {
    w.Close()                    // unblock the drainer
    <-done                       // drain pipe before swap
    os.Stderr = oldStderr        // restore stderr
    logger.SetLogLevel(oldLevel) // restore log level
})

// ... code that logs ...

// boilerplate: manual line-by-line substring scan
found := false
for _, line := range strings.Split(buf.String(), "\n") {
    if strings.Contains(line, "started") &&
        strings.Contains(line, "reqid=42") {
        found = true
        break
    }
}
if !found {
    t.Fatalf("expected log line not found:\n%s", buf.String())
}
```

</details>

<details>
<summary>With logger (log capturing in tests)</summary>

```go
import "github.com/voedger/voedger/pkg/goutils/logger"

// writers and log level are auto-restored when t ends - no Cleanup needed
cap := logger.StartCapture(t, logger.LogLevelInfo)

// ... code that logs ...
cap.HasLine("started", "reqid=42")
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
  - [WithContextAttrs: loggerctx.go#L25](loggerctx.go#L25)
  - [Ctx logging functions: loggerctx.go#L33](loggerctx.go#L33)
  - [sLogAttrsFromCtx: loggerctx.go#L153](loggerctx.go#L153)
  - **Predefined attribute key constants** ([consts.go](consts.go#L19))

    | Constant            | Key         | Example value |
    |---------------------|-------------|---------------|
    | `LogAttr_VApp`      | `vapp`      | `my.app`      |
    | `LogAttr_Feat`      | `feat`      | `magicmenu`   |
    | `LogAttr_ReqID`     | `reqid`     | `42`          |
    | `LogAttr_WSID`      | `wsid`      | `1001`        |
    | `LogAttr_Extension` | `extension` | `myFunc`      |

- **Stdlib log bridge** - Adapts byte-oriented stdlib `*log.Logger`
  (e.g. `http.Server.ErrorLog`) to context-aware `ErrorCtx` with
  optional line filtering
  - [NewStdlibLogBridge: loggerctx.go#L100](loggerctx.go#L100)
  - [WithFilter option: loggerctx.go#L112](loggerctx.go#L112)
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
  - [ILogCaptor: types.go#L29](types.go#L29)
  - [HasLine: logcapture.go#L30](logcapture.go#L30)
  - [EventuallyHasLine: logcapture.go#L41](logcapture.go#L41)
  - [NotContains: logcapture.go#L78](logcapture.go#L78)

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

