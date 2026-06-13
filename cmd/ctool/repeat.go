/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"github.com/spf13/cobra"
)

func newRepeatCmd() *cobra.Command {
	repeatCmd := &cobra.Command{
		Use:   "repeat",
		Short: "Execute the last incomplete command",
		RunE:  repeat,
	}

	if newCluster().Edition != clusterEditionN1 && !addSSHKeyFlag(repeatCmd) {
		return nil
	}

	return repeatCmd
}

func repeat(cmd *cobra.Command, _ []string) error {
	currentCmd = cmd
	cluster := newCluster()

	if cluster.Draft {
		return ErrClusterConfNotFound
	}

	if !cluster.existsNodeError() && (cluster.Cmd == nil || cluster.Cmd.isEmpty()) {
		return ErrNoIncompleteCommandWasFoundToRepeat
	}

	err := cluster.checkVersion()
	if err != nil {
		return err
	}

	defer saveClusterToJSON(cluster)

	if err := mkCommandDirAndLogFile(cmd, cluster); err != nil {
		return err
	}

	if err := cluster.Cmd.apply(cluster); err != nil {
		loggerError(err)
	}

	return err
}
