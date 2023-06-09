/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/cobrau"

	"github.com/untillpro/goutils/logger"
)

//go:embed version
var version string

// path to SSH key (flag --ssh-key)
var sshKey string
var red func(a ...interface{}) string
var green func(a ...interface{}) string

func main() {
	red = color.New(color.FgRed).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
	logger.PrintLine = printLogLine
	defer deleteScriptsTempDir()
	err := execRootCmd(os.Args, version)
	if err != nil {
		os.Exit(1)
	}
}

var rootCmd *cobra.Command

func execRootCmd(args []string, ver string) error {
	version = ver
	rootCmd = cobrau.PrepareRootCmd(
		"ctool",
		"Cluster managment utility",
		args,
		newVersionCmd(),
		newInitCmd(),
		newValidateCmd(),
		newUpgradeCmd(),
		newReplaceCmd(),
		newRepeatCmd(),
	)

	rootCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to SSH key")
	logger.SetLogLevel(getLoggerLevel())

	return cobrau.ExecCommandAndCatchInterrupt(rootCmd)
}

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Prints the version of the ctool utility",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("ctool version ", version)
		},
	}
	return cmd
}
