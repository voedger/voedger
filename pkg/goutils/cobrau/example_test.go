/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cobrau

import (
	"fmt"

	"github.com/spf13/cobra"
)

func Example() {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the service",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Service started")
		},
	}

	args := []string{"service", "start"}
	version := "2.1.0"
	rootCmd := PrepareRootCmd(
		"service",
		"Service management utility",
		args,
		version,
		startCmd,
	)

	if err := ExecCommandAndCatchInterrupt(rootCmd); err != nil {
		panic(err)
	}

	// Output:
	// Service started
}
