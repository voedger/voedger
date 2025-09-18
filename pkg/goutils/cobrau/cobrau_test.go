/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cobrau

import (
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu/require"
	"golang.org/x/sync/errgroup"
)

func TestPrepareRootCmd(t *testing.T) {
	require := require.New(t)

	t.Run("basic root command setup", func(t *testing.T) {
		testCmd := &cobra.Command{
			Use:   "test",
			Short: "Test command",
		}

		rootCmd := PrepareRootCmd(
			"myapp",
			"My test application",
			[]string{"myapp", "test"},
			"1.0.0",
			testCmd,
		)

		require.NotNil(rootCmd)
		require.Equal("myapp", rootCmd.Use)
		require.Equal("My test application", rootCmd.Short)
		require.True(rootCmd.SilenceUsage)
		require.True(rootCmd.SilenceErrors)
	})

	t.Run("version command is added automatically", func(t *testing.T) {
		rootCmd := PrepareRootCmd(
			"testapp",
			"Test app",
			[]string{"testapp", "version"},
			"2.1.0",
		)

		// Find version command
		versionCmd, _, err := rootCmd.Find([]string{"version"})
		require.NoError(err)
		require.NotNil(versionCmd)
		require.Equal("version", versionCmd.Use)
		require.Contains(versionCmd.Aliases, "ver")
	})

	t.Run("version command with alias 'ver'", func(t *testing.T) {
		rootCmd := PrepareRootCmd(
			"testapp",
			"Test app",
			[]string{"testapp", "ver"},
			"3.0.0",
		)

		// Find version command by alias
		versionCmd, _, err := rootCmd.Find([]string{"ver"})
		require.NoError(err)
		require.NotNil(versionCmd)
		require.Equal("version", versionCmd.Use)
	})

	t.Run("verbose and trace flags are added", func(t *testing.T) {
		rootCmd := PrepareRootCmd(
			"testapp",
			"Test app",
			[]string{"testapp"},
			"1.0.0",
		)

		verboseFlag := rootCmd.Flags().Lookup("verbose")
		require.NotNil(verboseFlag)
		require.Equal("v", verboseFlag.Shorthand)

		traceFlag := rootCmd.Flags().Lookup("trace")
		require.NotNil(traceFlag)
	})

	t.Run("subcommands are added correctly", func(t *testing.T) {
		cmd1 := &cobra.Command{Use: "start", Short: "Start service"}
		cmd2 := &cobra.Command{Use: "stop", Short: "Stop service"}

		rootCmd := PrepareRootCmd(
			"service",
			"Service manager",
			[]string{"service"},
			"1.0.0",
			cmd1,
			cmd2,
		)

		commands := rootCmd.Commands()
		commandNames := make([]string, 0, len(commands))
		for _, cmd := range commands {
			commandNames = append(commandNames, cmd.Use)
		}

		require.Contains(commandNames, "start")
		require.Contains(commandNames, "stop")
		require.Contains(commandNames, "version")
		require.Contains(commandNames, "completion")
	})

	t.Run("flags are added to subcommands recursively", func(t *testing.T) {
		subCmd := &cobra.Command{Use: "sub", Short: "Sub command"}
		nestedCmd := &cobra.Command{Use: "nested", Short: "Nested command"}
		subCmd.AddCommand(nestedCmd)

		_ = PrepareRootCmd(
			"testapp",
			"Test app",
			[]string{"testapp"},
			"1.0.0",
			subCmd,
		)

		// Check that flags are added to subcommand
		verboseFlag := subCmd.Flags().Lookup("verbose")
		require.NotNil(verboseFlag)

		traceFlag := subCmd.Flags().Lookup("trace")
		require.NotNil(traceFlag)

		// Check that flags are added to nested command
		nestedVerboseFlag := nestedCmd.Flags().Lookup("verbose")
		require.NotNil(nestedVerboseFlag)

		nestedTraceFlag := nestedCmd.Flags().Lookup("trace")
		require.NotNil(nestedTraceFlag)
	})
}

func TestLoggerIntegration(t *testing.T) {
	require := require.New(t)

	// Save original log level
	originalLevel := logger.SetLogLevel(logger.LogLevelInfo)
	defer logger.SetLogLevel(originalLevel)

	t.Run("verbose flag sets logger level", func(t *testing.T) {
		rootCmd := PrepareRootCmd(
			"testapp",
			"Test app",
			[]string{"testapp", "--verbose"},
			"1.0.0",
		)

		// Parse flags to set the flag values
		err := rootCmd.ParseFlags([]string{"--verbose"})
		require.NoError(err)

		// Execute PersistentPreRun
		rootCmd.PersistentPreRun(rootCmd, []string{})

		require.True(logger.IsVerbose())
	})

	t.Run("trace flag sets logger level", func(t *testing.T) {
		rootCmd := PrepareRootCmd(
			"testapp",
			"Test app",
			[]string{"testapp", "--trace"},
			"1.0.0",
		)

		// Parse flags to set the flag values
		err := rootCmd.ParseFlags([]string{"--trace"})
		require.NoError(err)

		// Execute PersistentPreRun
		rootCmd.PersistentPreRun(rootCmd, []string{})

		require.True(logger.IsTrace())
	})

	t.Run("trace flag takes precedence over verbose", func(t *testing.T) {
		rootCmd := PrepareRootCmd(
			"testapp",
			"Test app",
			[]string{"testapp", "--verbose", "--trace"},
			"1.0.0",
		)

		// Parse flags to set the flag values
		err := rootCmd.ParseFlags([]string{"--verbose", "--trace"})
		require.NoError(err)

		// Execute PersistentPreRun
		rootCmd.PersistentPreRun(rootCmd, []string{})

		require.True(logger.IsTrace())
	})
}

func TestExecCommandAndCatchInterrupt(t *testing.T) {
	require := require.New(t)

	t.Run("successful command execution", func(t *testing.T) {
		executed := false
		testCmd := &cobra.Command{
			Use: "test",
			Run: func(cmd *cobra.Command, args []string) {
				executed = true
			},
		}

		rootCmd := PrepareRootCmd(
			"testapp",
			"Test app",
			[]string{"testapp", "test"},
			"1.0.0",
			testCmd,
		)

		err := ExecCommandAndCatchInterrupt(rootCmd)
		require.NoError(err)
		require.True(executed)
	})

	t.Run("command execution with error", func(t *testing.T) {
		expectedErr := fmt.Errorf("test error")
		testCmd := &cobra.Command{
			Use: "test",
			RunE: func(cmd *cobra.Command, args []string) error {
				return expectedErr
			},
		}

		rootCmd := PrepareRootCmd(
			"testapp",
			"Test app",
			[]string{"testapp", "test"},
			"1.0.0",
			testCmd,
		)

		err := ExecCommandAndCatchInterrupt(rootCmd)
		require.Error(err)
		require.Equal(expectedErr, err)
	})

	t.Run("command with PreRun and PostRun hooks", func(t *testing.T) {
		var hooks []string

		testCmd := &cobra.Command{
			Use: "test",
			PreRun: func(cmd *cobra.Command, args []string) {
				hooks = append(hooks, "pre")
			},
			Run: func(cmd *cobra.Command, args []string) {
				hooks = append(hooks, "run")
			},
			PostRun: func(cmd *cobra.Command, args []string) {
				hooks = append(hooks, "post")
			},
		}

		rootCmd := PrepareRootCmd(
			"testapp",
			"Test app",
			[]string{"testapp", "test"},
			"1.0.0",
			testCmd,
		)

		err := ExecCommandAndCatchInterrupt(rootCmd)
		require.NoError(err)
		require.Equal([]string{"pre", "run", "post"}, hooks)
	})
}

func TestInterrupt(t *testing.T) {
	require := require.New(t)
	interruptSignals := []os.Signal{
		os.Interrupt,
		syscall.SIGTERM,
	}

	for _, interruptSignal := range interruptSignals {
		t.Run(interruptSignal.String(), func(t *testing.T) {
			cmdRunStart := make(chan interface{})
			var signalsCh chan os.Signal
			longRunningCmd := &cobra.Command{
				Use: "long",
				Run: func(cmd *cobra.Command, args []string) {
					signalsCh = cmd.Context().Value(signalChKey).(chan os.Signal)
					close(cmdRunStart)
					select {
					case <-cmd.Context().Done():
						// Context was cancelled due to interrupt signal
					case <-time.After(5 * time.Second):
						// This should not happen in our test
						t.Error("Command should have been interrupted")
					}
				},
			}
			rootCmd := PrepareRootCmd(
				"testapp",
				"Test app",
				[]string{"testapp", "long"},
				"1.0.0",
				longRunningCmd,
			)

			g := errgroup.Group{}
			g.Go(func() error {
				return ExecCommandAndCatchInterrupt(rootCmd)
			})
			<-cmdRunStart
			signalsCh <- interruptSignal
			require.NoError(g.Wait())
		})
	}
}

func TestAddFlagsToCommands(t *testing.T) {
	require := require.New(t)

	t.Run("flags added to root command", func(t *testing.T) {
		rootCmd := &cobra.Command{Use: "root"}
		addFlagsToCommands(rootCmd)

		verboseFlag := rootCmd.Flags().Lookup("verbose")
		require.NotNil(verboseFlag)
		require.Equal("v", verboseFlag.Shorthand)
		require.Equal("Enable verbose output", verboseFlag.Usage)

		traceFlag := rootCmd.Flags().Lookup("trace")
		require.NotNil(traceFlag)
		require.Equal("Enable extremely verbose output", traceFlag.Usage)

		require.True(rootCmd.SilenceErrors)
	})

	t.Run("flags added recursively to subcommands", func(t *testing.T) {
		rootCmd := &cobra.Command{Use: "root"}
		subCmd1 := &cobra.Command{Use: "sub1"}
		subCmd2 := &cobra.Command{Use: "sub2"}
		nestedCmd := &cobra.Command{Use: "nested"}

		rootCmd.AddCommand(subCmd1, subCmd2)
		subCmd1.AddCommand(nestedCmd)

		addFlagsToCommands(rootCmd)

		// Check root command
		require.NotNil(rootCmd.Flags().Lookup("verbose"))
		require.NotNil(rootCmd.Flags().Lookup("trace"))

		// Check subcommands
		require.NotNil(subCmd1.Flags().Lookup("verbose"))
		require.NotNil(subCmd1.Flags().Lookup("trace"))
		require.NotNil(subCmd2.Flags().Lookup("verbose"))
		require.NotNil(subCmd2.Flags().Lookup("trace"))

		// Check nested command
		require.NotNil(nestedCmd.Flags().Lookup("verbose"))
		require.NotNil(nestedCmd.Flags().Lookup("trace"))
	})
}
