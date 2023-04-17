/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 */

package cmd

import "fmt"

func Execute(version string) int {
	edgerVersion = version

	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		return 1
	}
	return 0
}
