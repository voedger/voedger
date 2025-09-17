# exec

Simplifies command execution and piping in Go with fluent API for
chaining commands and capturing output.

## Problem

Standard Go command execution requires verbose setup, manual pipe
management, and complex error handling for command chains.

<details>
<summary>Without exec</summary>

```go
// Executing "printf '1\n2\n3' | grep 2" manually
cmd1 := exec.Command("printf", "1\\n2\\n3")
cmd2 := exec.Command("grep", "2")

// Manual pipe setup - boilerplate here
r, w := io.Pipe()
cmd1.Stdout = w
cmd2.Stdin = r

// Complex error handling and coordination
var stdout, stderr bytes.Buffer
cmd2.Stdout = &stdout
cmd2.Stderr = &stderr

// Start commands in correct order - easy to mess up
if err := cmd1.Start(); err != nil {
    return "", "", err
}
if err := cmd2.Start(); err != nil {
    return "", "", err
}

// Close writer after first command starts
go func() {
    defer w.Close()
    cmd1.Wait()
}()

// Wait for second command
if err := cmd2.Wait(); err != nil {
    return "", "", err
}

result := strings.TrimSpace(stdout.String())
```
</details>

<details>
<summary>With exec</summary>

```go
import "github.com/voedger/voedger/pkg/goutils/exec"

// Same functionality in one fluent chain
stdout, stderr, err := new(exec.PipedExec).
    Command("printf", "1\\n2\\n3").
    Command("grep", "2").
    RunToStrings()
```
</details>

## Features

- **[Command chaining](exec.go#L67)** - Fluent API for building
  command pipes
  - [Pipe setup automation: exec.go#L45](exec.go#L45)
  - [Sequential command execution: exec.go#L108](exec.go#L108)
- **[Output capture](exec.go#L125)** - Capture stdout/stderr to
  strings or writers
  - [Concurrent output reading: exec.go#L140](exec.go#L140)
  - [Synchronized completion: exec.go#L165](exec.go#L165)
- **[Working directory](exec.go#L77)** - Set working directory for
  commands
- **[Process management](exec.go#L83)** - Start and wait for command
  completion
  - [Error propagation: exec.go#L88](exec.go#L88)

## Use

See [example](example_test.go)
