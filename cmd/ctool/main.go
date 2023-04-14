/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package main

import (
	"os"

	_ "embed"

	"github.com/untillpro/goutils/cobrau"
)

//go:embed version
var version string

func main() {
	if err := execRootCmd(os.Args, version); err != nil {
		os.Exit(1)
	}
}

func execRootCmd(args []string, ver string) error {
	rootCmd := cobrau.PrepareRootCmd(
		"ctool",
		"",
		args,
		ver,
		newClusterCmd(),
	)

	return cobrau.ExecCommandAndCatchInterrupt(rootCmd)
}
