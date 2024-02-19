/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package main

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/cobrau"

	"github.com/untillpro/goutils/logger"
)

//go:embed version
var version string

// path to SSH key (flag --ssh-key)
var sshKey string

// SSH port
var sshPort string

// skip checking nodes for the presence of the minimum allowable amount of RAM
var skipNodeMemoryCheck bool

var acmeDomains string

var devMode bool

var red func(a ...interface{}) string
var green func(a ...interface{}) string

// nolint
func main() {

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	dir := filepath.Dir(ex)
	os.Chdir(dir)

	red = color.New(color.FgRed).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
	logger.PrintLine = printLogLine
	prepareScripts()
	defer func() {
		err := deleteScriptsTempDir()
		if err != nil {
			loggerError(err.Error())
		}
	}()
	err = execRootCmd(os.Args, version)
	if err != nil {
		loggerError(err.Error())
		os.Exit(1)
	}
}

var rootCmd *cobra.Command

// nolint
func execRootCmd(args []string, ver string) error {
	version = ver
	rootCmd = cobrau.PrepareRootCmd(
		"ctool",
		"Cluster managment utility",
		args,
		version,
		newInitCmd(),
		newValidateCmd(),
		newUpgradeCmd(),
		newReplaceCmd(),
		newRepeatCmd(),
		newBackupCmd(),
		newAcmeCmd(),
		newRestoreCmd(),
	)
	rootCmd.SilenceErrors = true
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Perform a dry run of the command without making any actual changes")

	rootCmd.PersistentFlags().BoolVar(&skipNodeMemoryCheck, "skip-node-memory-check", false, "Skip checking nodes for the presence of the minimum allowable amount of RAM")
	rootCmd.PersistentFlags().BoolVar(&devMode, "dev-mode", false, "Use development mode for DB stack")
	logger.SetLogLevel(getLoggerLevel())

	return cobrau.ExecCommandAndCatchInterrupt(rootCmd)
}
