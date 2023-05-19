/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */
package main

import (
	"github.com/spf13/cobra"
)

func newRepairCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repair",
		Short: "Repairs the cluster after a failed deployment",
		RunE:  repair,
	}
}

func repair(cmd *cobra.Command, arg []string) error {

	err := mkCommandDirAndLogFile(cmd)
	if err != nil {
		return err
	}

	return nil
}
