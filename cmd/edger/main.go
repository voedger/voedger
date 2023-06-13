/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author: Nikolay Nikitin
 * @author: Alisher Nurmanov
 */

package main

import (
	_ "embed"
	"os"

	"fmt"
	"runtime/debug"


	"github.com/untillpro/goutils/cobrau"
)

//go:embed version
var version string

func main() {

	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("No build info")
		return
	}
	for _, mod := range info.Deps {
		fmt.Printf("path: %s version: %s\n", mod.Path, mod.Version)
	}

	if err := execRootCmd(os.Args, version); err != nil {
		os.Exit(1)
	}
}

func execRootCmd(args []string, ver string) error {
	version = ver

	rootCmd := cobrau.PrepareRootCmd(
		"edger",
		"",
		args,
		version,
		newServerCmd(),
		newRunEdgerCmd(),
	)
	//rootCmd.PersistentFlags().BoolVar(&internal.IsDryRun, "dry-run", false, "Simulate the execution of the command without actually modifying any files or data")

	// Can just use `return rootCmd.Execute()`
	return cobrau.ExecCommandAndCatchInterrupt(rootCmd)
}
