/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
)

func newRepeatCmd() *cobra.Command {
	repeatCmd := &cobra.Command{
		Use:   "repeat",
		Short: "executing the last incomplete command",
		RunE:  repeat,
	}

	repeatCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to SSH key")
	if err := repeatCmd.MarkPersistentFlagRequired("ssh-key"); err != nil {
		logger.Error(err.Error())
		return nil
	}

	return repeatCmd
}

func repeat(cmd *cobra.Command, arg []string) error {
	cluster, err := newCluster()
	if err != nil {
		return err
	}

	// nolint
	defer cluster.saveToJSON()

	if !cluster.existsNodeError() && (cluster.Cmd == nil || cluster.Cmd.isEmpty()) {
		logger.Info("no active command found to repeat")
		return nil
	}

	// nolint
	mkCommandDirAndLogFile(cmd, cluster)

	if err = cluster.Cmd.apply(cluster); err != nil {
		logger.Error(err)
	}

	return err
}
