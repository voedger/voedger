/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
)

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validates the configuration and status of the cluster for errors",
		RunE:  validate,
	}
}

func validate(cmd *cobra.Command, arg []string) error {

	cluster := newCluster()

	// nolint
	mkCommandDirAndLogFile(cmd, cluster)

	if !cluster.exists {
		logger.Error(red(ErrClusterConfNotFound.Error()))
		return ErrClusterConfNotFound
	}

	err := cluster.validate()
	if err == nil {
		logger.Info(green("cluster configuration is ok"))
	}
	return err
}
