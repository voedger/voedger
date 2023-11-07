/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */
package main

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
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
	// nolint
	defer cluster.saveToJSON()

	c := newCmd(ckReplace, strings.Join(arg, " "))
	if err := cluster.applyCmd(c); err != nil {
		logger.Error(err.Error())
		return err
	}

	err := mkCommandDirAndLogFile(cmd, cluster)
	if err != nil {
		return err
	}

	err = cluster.validate()
	if err == nil {
		println("cluster configuration is ok")
		if err = cluster.Cmd.apply(cluster); err != nil {
			logger.Error(err)
		}
	}

	return err
}
