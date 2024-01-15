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
	"github.com/untillpro/goutils/cobrau"
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
	rootCmd := cobrau.PrepareRootCmd(
		"vpm",
		"",
		args,
		ver,
		newCompileCmd(),
		newBaselineCmd(),
		newCompatCmd(),
	)
	rootCmd.InitDefaultHelpCmd()
	rootCmd.InitDefaultCompletionCmd()
	correctCommandTexts(rootCmd)
	return cobrau.ExecCommandAndCatchInterrupt(rootCmd)
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
