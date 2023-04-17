/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package main

import (
	_ "embed"
	"os"

	"github.com/untillpro/voedger/cmd/edger/internal/cmd"
)

//go:embed version
var version string

func main() {
	os.Exit(cmd.Execute(version))
}
