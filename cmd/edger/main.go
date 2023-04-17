/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
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
