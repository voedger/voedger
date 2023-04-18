/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
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
