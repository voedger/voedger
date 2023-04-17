/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "edger",
		Short: "",
	}

	cmd.CompletionOptions.DisableDefaultCmd = true
	// cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "detailed logging of the command execution process")

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newServerCmd())

	return &cmd
}
