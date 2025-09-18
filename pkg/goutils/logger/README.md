# Logger

Structured logging with hierarchical levels and automatic caller
information. Provides thread-safe logging with customizable output
formatting and level-based filtering.

## Problem

Go's standard log package lacks level-based filtering and structured
output formatting, forcing developers to implement verbose logging
infrastructure manually.

<details>
<summary>Without logger</summary>

```go
// Manual level checking and formatting - error-prone boilerplate
var logLevel int = 3 // Info level
var mu sync.Mutex

func logError(msg string, args ...interface{}) {
    if logLevel >= 1 { // boilerplate: manual level check
        mu.Lock() // boilerplate: manual thread safety
        defer mu.Unlock()
        pc, file, line, _ := runtime.Caller(1) // boilerplate: stack trace
        fn := runtime.FuncForPC(pc)
        funcName := "unknown"
        if fn != nil {
            funcName = fn.Name() // common mistake: not handling nil
        }
        timestamp := time.Now().Format("01/02 15:04:05.000")
        fmt.Fprintf(os.Stderr, "%s: ERROR: [%s:%d]: %s",
            timestamp, funcName, line, fmt.Sprintf(msg, args...))
    }
}

func logInfo(msg string, args ...interface{}) {
    if logLevel >= 3 { // boilerplate: repeated level logic
        mu.Lock()
        defer mu.Unlock()
        // boilerplate: duplicate formatting code
        pc, file, line, _ := runtime.Caller(1)
        fn := runtime.FuncForPC(pc)
        funcName := "unknown"
        if fn != nil {
            funcName = fn.Name()
        }
        timestamp := time.Now().Format("01/02 15:04:05.000")
        fmt.Fprintf(os.Stdout, "%s: INFO: [%s:%d]: %s",
            timestamp, funcName, line, fmt.Sprintf(msg, args...))
    }
}

func expensiveOperation() string {
    // common mistake: always computing expensive debug info
    return strings.Join(getAllDebugInfo(), ", ")
}

func main() {
    logError("Connection failed: %s", "timeout")
    logInfo("Debug info: %s", expensiveOperation()) // always computed
}
```

</details>

<details>
<summary>With logger</summary>

```go
import "github.com/voedger/voedger/pkg/goutils/logger"

func main() {
    logger.Error("Connection failed:", "timeout")

    // Conditional logging avoids expensive operations
    if logger.IsVerbose() {
        logger.Verbose("Debug info:", expensiveOperation())
    }

    // Temporary level changes with automatic restore
    defer logger.SetLogLevelWithRestore(logger.LogLevelTrace)()
    logger.Trace("Detailed execution flow")
}
```

</details>

## Features

- **[Level filtering](logger.go#L22)** - Hierarchical log levels with
  atomic level switching
  - [Level constants: logger.go#L22](logger.go#L22)
  - [Atomic level checking: impl.go#L37](impl.go#L37)
  - [Level restoration: logger.go#L36](logger.go#L36)
- **[Caller tracking](impl.go#L42)** - Automatic function name and line
  number extraction
  - [Stack frame analysis: impl.go#L42](impl.go#L42)
  - [Formatted output generation: impl.go#L55](impl.go#L55)
  - [Skip frame control: logger.go#L64](logger.go#L64)
- **[Output customization](logger.go#L88)** - Pluggable output handlers
  with stderr/stdout routing
  - [Custom PrintLine function: logger.go#L88](logger.go#L88)
  - [Default output routing: logger.go#L90](logger.go#L90)
- **[Performance optimization](logger.go#L68)** - Level guards prevent
  expensive operations
  - [Conditional logging checks: logger.go#L68](logger.go#L68)

## Use

See [basic usage test](logger_test.go#L25)

## Example Output

```
09/29 13:29:04.355: *****: [core-logger.Test_BasicUsage:22]: Hello world arg1 arg2
09/29 13:29:04.373: !!!: [core-logger.Test_BasicUsage:23]: My warning
09/29 13:29:04.374: ===: [core-logger.Test_BasicUsage:24]: My info
09/29 13:29:04.374: ---: [core-logger.Test_BasicUsage:35]: Now you should see my Trace
09/29 13:29:04.374: !!!: [core-logger.Test_BasicUsage:41]: You should see my warning
09/29 13:29:04.374: !!!: [core-logger.Test_BasicUsage:42]: You should see my info
09/29 13:29:04.374: *****: [core-logger.(*mystruct).iWantToLog:55]: OOPS
```

## Links

- [Why does the TRACE level exist, and when should I use it rather than DEBUG?](https://softwareengineering.stackexchange.com/questions/279690/why-does-the-trace-level-exist-and-when-should-i-use-it-rather-than-debug)
  - [Good answer](https://softwareengineering.stackexchange.com/questions/279690/why-does-the-trace-level-exist-and-when-should-i-use-it-rather-than-debug/360810#360810)