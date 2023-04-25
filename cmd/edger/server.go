/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package main

import (
	"github.com/spf13/cobra"
)

func newServerCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "server",
		Short: "Runs edger server cycle",
		Run: func(*cobra.Command, []string) {

		},
	}

	return &cmd
}
