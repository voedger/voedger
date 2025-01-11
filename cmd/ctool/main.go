/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package main

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/voedger/voedger/pkg/goutils/cobrau"

	"github.com/voedger/voedger/pkg/goutils/logger"
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

var mu sync.Mutex

// nolint
func main() {

	cursorOff()
	defer cursorOn()
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
		cursorOn()
		os.Exit(1)
	}

}

// adds to the command flag --ssh-key
// If the environment variable VOEDGER_SSH_KEY is not established, then the flag is marked as a required
func addSshKeyFlag(cmds ...*cobra.Command) bool {
	for _, cmd := range cmds {
		cmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to SSH key")
		value, exists := os.LookupEnv(envVoedgerSshKey)
		if !exists || value == "" {
			if err := cmd.MarkPersistentFlagRequired("ssh-key"); err != nil {
				loggerError(err.Error())
				return false
			}
		}
	}
	return true
}

// nolint
func cursorOff() {
	cmd := exec.Command("setterm", "--cursor", "off")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// nolint
func cursorOn() {
	cmd := exec.Command("setterm", "--cursor", "on")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

var (
	rootCmd    *cobra.Command
	currentCmd *cobra.Command
)

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
		newMonCmd(),
		newAlertCmd(),
	)

	rootCmd.PersistentFlags().BoolP("help", "h", false, "Display help for the command")

	rootCmd.SilenceErrors = true
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Perform a dry run without making any changes")

	rootCmd.PersistentFlags().BoolVar(&skipNodeMemoryCheck, "skip-node-memory-check", false, "Skip the minimum RAM check for nodes")
	rootCmd.PersistentFlags().BoolVar(&devMode, "dev-mode", false, "Use development mode for the database stack")
	logger.SetLogLevel(getLoggerLevel())

	return cobrau.ExecCommandAndCatchInterrupt(rootCmd)
}
