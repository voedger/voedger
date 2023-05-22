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
	cmd := &cobra.Command{
		Use:   "repeat",
		Short: "executing the last incomplete command",
		RunE:  repeat,
	}

	return cmd
}

func repeat(cmd *cobra.Command, arg []string) error {
	cluster := newCluster()
	defer cluster.saveToJSON()

	if !cluster.existsNodeError() && cluster.Cmd.isEmpty() {
		logger.Info("no active command found to repeat")
		return nil
	}

	mkCommandDirAndLogFile(cmd, cluster)

	var err error

	if err = cluster.Cmd.apply(cluster); err != nil {
		logger.Error(err)
	}

	return err
}
