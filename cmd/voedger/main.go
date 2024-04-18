/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 * @author Maxim Geraskin
 */

package main

import (
	_ "embed"

	"os"

	"github.com/voedger/voedger/pkg/goutils/cobrau"
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
		"ce",
		"Voedger server Community Edition",
		args,
		ver,
		newServerCmd(),
	)

	return cobrau.ExecCommandAndCatchInterrupt(rootCmd)
}
