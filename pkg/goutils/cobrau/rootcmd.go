/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package cobrau

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

/*

Persistent flags:

  -v, --verbose   Print verbose output (detailed level)
      --trace     Print trace output   (most detailed level)

*/

func PrepareRootCmd(use string, short string, args []string, version string, cmds ...*cobra.Command) *cobra.Command {

	var rootCmd = &cobra.Command{
		Use:   use,
		Short: short,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if ok, _ := cmd.Flags().GetBool("trace"); ok {
				logger.SetLogLevel(logger.LogLevelTrace)
				logger.Verbose("Using logger.LogLevelTrace...")
			} else if ok, _ := cmd.Flags().GetBool("verbose"); ok {
				logger.SetLogLevel(logger.LogLevelVerbose)
				logger.Verbose("Using logger.LogLevelVerbose...")
			}
		},
	}

	var versionCmd = &cobra.Command{
		Use:     "version",
		Short:   "Print the current version",
		Aliases: []string{"ver"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}

	rootCmd.SetArgs(args[1:])
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(cmds...)
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().Bool("trace", false, "Enable extremely verbose output")
	rootCmd.SilenceUsage = true
	return rootCmd
}
