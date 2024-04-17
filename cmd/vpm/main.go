/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/voedger/voedger/pkg/goutils/cobrau"
)

//go:embed version
var version string

func main() {
	if err := execRootCmd(os.Args, version); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func execRootCmd(args []string, ver string) error {
	params := &vpmParams{}
	rootCmd := cobrau.PrepareRootCmd(
		"vpm",
		"",
		args,
		ver,
		newCompileCmd(params),
		newBaselineCmd(params),
		newCompatCmd(params),
		newOrmCmd(params),
		newInitCmd(params),
		newTidyCmd(params),
	)
	rootCmd.InitDefaultHelpCmd()
	rootCmd.InitDefaultCompletionCmd()
	correctCommandTexts(rootCmd)
	initGlobalFlags(rootCmd, params)
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return prepareParams(cmd, params, args)
	}
	setNoArgs(rootCmd)
	return cobrau.ExecCommandAndCatchInterrupt(rootCmd)
}

func setNoArgs(cmd *cobra.Command) {
	if cmd.Args == nil {
		cmd.Args = exactArgs(0)
	}
	for _, subCmd := range cmd.Commands() {
		setNoArgs(subCmd)
	}
}

// correctCommandTexts makes first letter of command and its flags descriptions small
// works recursively for all subcommands
func correctCommandTexts(cmd *cobra.Command) {
	correctCommandFlagTexts(cmd)
	for _, c := range cmd.Commands() {
		c.Short = makeFirstLetterSmall(c.Short)
		correctCommandTexts(c)
	}
}

func correctCommandFlagTexts(cmd *cobra.Command) {
	correctFlagSetTexts(cmd.Flags())
	correctFlagSetTexts(cmd.PersistentFlags())
}

func correctFlagSetTexts(fs *pflag.FlagSet) {
	fs.VisitAll(func(f *pflag.Flag) {
		f.Usage = makeFirstLetterSmall(f.Usage)
	})
}

func makeFirstLetterSmall(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[0:1]) + s[1:]
}

func initGlobalFlags(cmd *cobra.Command, params *vpmParams) {
	cmd.SilenceErrors = true
	cmd.PersistentFlags().StringVarP(&params.Dir, "change-dir", "C", "", "change to dir before running the command. Any files named on the command line are interpreted after changing directories")
}

func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		runHelpFuncInstead := func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		}
		switch {
		case len(args) == 1 && args[0] == "help":
			cmd.RunE = runHelpFuncInstead
			return nil
		case n == 0 && len(args) > 0:
			return fmt.Errorf("'%s' accepts no argument. Run '%s help'", cmd.CommandPath(), cmd.CommandPath())
		case len(args) != n:
			cmd.RunE = runHelpFuncInstead
			return nil
		}
		return nil
	}
}
