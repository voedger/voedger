/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */
package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/untillpro/goutils/logger"
)

func newReplaceCmd() *cobra.Command {
	replaceCmd := &cobra.Command{
		Use:   "replace",
		Short: "Replaces the cluster node",
		RunE:  replace,
	}
	replaceCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to SSH key")
	if err := replaceCmd.MarkPersistentFlagRequired("ssh-key"); err != nil {
		logger.Error(err.Error())
		return nil
	}

	return replaceCmd

}

func replace(cmd *cobra.Command, arg []string) error {

	cluster, err := newCluster()
	if err != nil {
		return err
	}

	// nolint
	defer cluster.saveToJSON()

	c := newCmd(ckReplace, strings.Join(arg, " "))
	if err := cluster.applyCmd(c); err != nil {
		return err
	}

	err = mkCommandDirAndLogFile(cmd, cluster)
	if err != nil {
		return err
	}

	replacedAddress := cluster.Cmd.args()[0]

	err = cluster.validate()
	if err == nil {
		println("cluster configuration is ok")
		if err = cluster.Cmd.apply(cluster); err != nil {
			return err
		}
	}

	fmt.Println("Replaced address:", replacedAddress)
	cluster.ReplacedAddresses = append(cluster.ReplacedAddresses, replacedAddress)
	return nil
}
