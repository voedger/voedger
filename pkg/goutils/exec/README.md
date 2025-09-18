# Exec

Unix-style command pipeline builder with fluent API for chaining shell
commands and automatic pipe management.

## Problem

Building command pipelines in Go requires manually connecting stdout/stdin
pipes between processes, handling goroutines for concurrent execution,
and managing complex error propagation across the chain.

<details>
<summary>Without exec</summary>

```go
// Manual pipe setup for: echo "data" | grep "data" | wc -l
cmd1 := exec.Command("echo", "data")
cmd2 := exec.Command("grep", "data")
cmd3 := exec.Command("wc", "-l")

// Create pipes manually - boilerplate here
pipe1, err := cmd1.StdoutPipe()
if err != nil {
    return err // Common mistake: forgetting error checks
}
cmd2.Stdin = pipe1

pipe2, err := cmd2.StdoutPipe()
if err != nil {
    return err
}
cmd3.Stdin = pipe2

// Start all commands in correct order - easy to mess up
if err := cmd1.Start(); err != nil {
    return err
}
if err := cmd2.Start(); err != nil {
    return err
}
if err := cmd3.Start(); err != nil {
    return err
}

// Wait for completion and handle errors - more boilerplate
var firstErr error
if err := cmd1.Wait(); err != nil && firstErr == nil {
    firstErr = err
}
if err := cmd2.Wait(); err != nil && firstErr == nil {
    firstErr = err
}
if err := cmd3.Wait(); err != nil && firstErr == nil {
    firstErr = err
}
return firstErr
```
</details>

<details>
<summary>With exec</summary>

```go
import "github.com/voedger/voedger/pkg/goutils/exec"

// Simple pipeline: echo "data" | grep "data" | wc -l
stdout, stderr, err := new(exec.PipedExec).
    Command("echo", "data").
    Command("grep", "data").
    Command("wc", "-l").
    RunToStrings()

// With working directory and output redirection
err = new(exec.PipedExec).
    Command("tinygo", "build", "-o", "app.wasm").
    WorkingDir("/project/dir").
    Run(os.Stdout, os.Stderr)
```
</details>

## Features

- **[Pipeline builder](exec.go#L32)** - Fluent API for command chaining
  - [Command chaining: exec.go#L48](exec.go#L48)
  - [Automatic pipe connection: exec.go#L53](exec.go#L53)
  - [Working directory support: exec.go#L76](exec.go#L76)
- **Execution modes** - Multiple output handling strategies
  - [Stream to writers: exec.go#L120](exec.go#L120)
  - [Capture to strings: exec.go#L129](exec.go#L129)
  - [Async start/wait: exec.go#L95](exec.go#L95)
- **Process control** - Advanced process management
  - [Command access: exec.go#L66](exec.go#L66)
  - [Error propagation: exec.go#L83](exec.go#L83)

## Use

See [example](example_test.go)
