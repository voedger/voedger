/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */
package main

import (
	"github.com/spf13/cobra"
)

func newReplaceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "replace",
		Short: "Replaces the cluster node",
		RunE:  replace,
	}
}

func replace(cmd *cobra.Command, arg []string) error {

	cluster := newCluster()
	defer cluster.saveToJSON()

	err := mkCommandDirAndLogFile(cmd)
	if err != nil {
		return err
	}

	return err
}
