/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
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
