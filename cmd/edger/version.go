/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var edgerVersion string

func newVersionCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "version",
		Short: "Prints the version of the edger utility",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("edger version ", edgerVersion)
		},
	}
	return &cmd
}
