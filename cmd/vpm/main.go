/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

import (
	_ "embed"
	"fmt"
	"os"

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
		"voedger package manager",
		args,
		ver,
		newCompileCmd(),
	)

	return cobrau.ExecCommandAndCatchInterrupt(rootCmd)
}
