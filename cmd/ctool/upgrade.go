/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUpgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Update the cluster version to the current one",
		RunE:  upgrade,
	}
}

func upgrade(cmd *cobra.Command, arg []string) error {

	cluster := newCluster()

	if cluster.ActualVersion == cluster.DesiredVersion {
		fmt.Println("no update required")
		return nil
	}

	err := mkCommandDirAndLogFile(cmd)
	if err != nil {
		return err
	}

	return nil
}
