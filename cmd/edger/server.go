/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author: Nikolay Nikitin
 * @author: Alisher Nurmanov
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
