# cobrau

Standardized Cobra CLI application setup with integrated logging,
signal handling, and common command patterns.

## Problem

Setting up Cobra CLI applications requires repetitive boilerplate for
flags, version commands, signal handling, and logger integration.

<details>
<summary>Without cobrau</summary>

```go
import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "sync"

    "github.com/spf13/cobra"
    "github.com/voedger/voedger/pkg/goutils/logger"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "myapp",
        Short: "My application",
        PersistentPreRun: func(cmd *cobra.Command, args []string) {
            // Manual flag handling - boilerplate here
            if ok, _ := cmd.Flags().GetBool("trace"); ok {
                logger.SetLogLevel(logger.LogLevelTrace)
                logger.Verbose("Using logger.LogLevelTrace...")
            } else if ok, _ := cmd.Flags().GetBool("verbose"); ok {
                logger.SetLogLevel(logger.LogLevelVerbose)
                logger.Verbose("Using logger.LogLevelVerbose...")
            }
        },
    }

    // Manual flag setup for every command - repetitive
    rootCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
    rootCmd.Flags().Bool("trace", false, "Enable trace output")
    rootCmd.InitDefaultHelpFlag()
    rootCmd.SilenceErrors = true

    // Manual version command creation
    versionCmd := &cobra.Command{
        Use:   "version",
        Short: "Print version",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Printf("%s version %s\n", cmd.Root().Name(), version)
        },
    }

    rootCmd.SetArgs(os.Args[1:])
    rootCmd.AddCommand(versionCmd)
    rootCmd.InitDefaultCompletionCmd()
    rootCmd.SilenceUsage = true

    // Manual signal handling - complex and error-prone
    signals := make(chan os.Signal, 1)
    ctx, cancel := context.WithCancel(context.Background())
    signal.Notify(signals, os.Interrupt)

    var wg sync.WaitGroup
    var err error
    wg.Add(1)
    go func() {
        defer wg.Done()
        err = rootCmd.ExecuteContext(ctx)
        cancel()
    }()

    select {
    case <-signals:
        logger.Info("signal received")
        cancel()
    case <-ctx.Done():
    }
    wg.Wait()

    if err != nil {
        os.Exit(1)
    }
}
```
</details>

<details>
<summary>With cobrau</summary>

```go
import (
    "os"
    "github.com/voedger/voedger/pkg/goutils/cobrau"
)

func main() {
    rootCmd := cobrau.PrepareRootCmd(
        "myapp",
        "My application",
        os.Args,
        "1.0.0",
        newServerCmd(),
        newConfigCmd(),
    )

    if err := cobrau.ExecCommandAndCatchInterrupt(rootCmd); err != nil {
        os.Exit(1)
    }
}
```
</details>

## Features

- **Root command setup** - Complete CLI application initialization
  - [Command structure creation: rootcmd.go#L35](rootcmd.go#L35)
  - [Automatic flag integration: rootcmd.go#L24](rootcmd.go#L24)
  - [Logger level configuration: rootcmd.go#L40](rootcmd.go#L40)
- **Signal handling** - Graceful interrupt and termination support
  - [Context-based execution: catch_interrupt.go#L18](catch_interrupt.go#L18)
  - [Signal capture logic: catch_interrupt.go#L29](catch_interrupt.go#L29)

## Use

See [example usage](example_test.go)
