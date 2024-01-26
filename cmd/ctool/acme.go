/*
* Copyright (c) 2024-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"github.com/spf13/cobra"
)

// nolint
func newAcmeCmd() *cobra.Command {
	acmeAddCmd := &cobra.Command{
		Use:   "add [<domain1,domain2...>]",
		Short: "Adds one or more domains to the acme domain list",
		Args:  cobra.ExactArgs(1),
		RunE:  acmeAdd,
	}

	acmeAddCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to SSH key")
	if err := acmeAddCmd.MarkPersistentFlagRequired("ssh-key"); err != nil {
		loggerError(err.Error())
		return nil
	}

	acmeListCmd := &cobra.Command{
		Use:   "list",
		Short: "Displaying a list of ACME domains",
		RunE:  acmeList,
	}

	acmeRemoveCmd := &cobra.Command{
		Use:   "remove [<domain1,domain2...>]",
		Short: "Removes one or more domains from the acme domain list",
		Args:  cobra.ExactArgs(1),
		RunE:  acmeRemove,
	}
	acmeRemoveCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to SSH key")
	if err := acmeRemoveCmd.MarkPersistentFlagRequired("ssh-key"); err != nil {
		loggerError(err.Error())
		return nil
	}

	acmeCmd := &cobra.Command{
		Use:   "acme",
		Short: "ACME settings",
	}

	acmeCmd.AddCommand(acmeAddCmd, acmeListCmd, acmeRemoveCmd)

	return acmeCmd

}

func acmeAdd(cmd *cobra.Command, args []string) error {
	cluster := newCluster()

	if !cluster.clusterConfigFileExists() {
		return ErrClusterConfNotFound
	}

	c := newCmd(ckAcme, append([]string{"add"}, args...))
	if err := cluster.applyCmd(c); err != nil {
		loggerError(err.Error())
		return err
	}

	defer func(cluster *clusterType) {
		err := cluster.saveToJSON()
		if err != nil {
			loggerError(err.Error())
		}
	}(cluster)

	if err := mkCommandDirAndLogFile(cmd, cluster); err != nil {
		return err
	}

	if err := cluster.Cmd.apply(cluster); err != nil {
		loggerError(err)
		return err
	}

	return nil
}

func acmeRemove(cmd *cobra.Command, args []string) error {
	cluster := newCluster()

	if !cluster.clusterConfigFileExists() {
		return ErrClusterConfNotFound
	}

	c := newCmd(ckAcme, append([]string{"remove"}, args...))
	if err := cluster.applyCmd(c); err != nil {
		loggerError(err.Error())
		return err
	}

	defer func(cluster *clusterType) {
		err := cluster.saveToJSON()
		if err != nil {
			loggerError(err.Error())
		}
	}(cluster)

	if err := mkCommandDirAndLogFile(cmd, cluster); err != nil {
		return err
	}

	if err := cluster.Cmd.apply(cluster); err != nil {
		loggerError(err)
		return err
	}

	return nil
}

func acmeList(cmd *cobra.Command, args []string) error {
	cluster := newCluster()

	if !cluster.clusterConfigFileExists() {
		return ErrClusterConfNotFound
	}

	if len(cluster.Acme.domains()) == 0 {
		loggerInfo("ACME domains list is empty")
		return nil
	}

	loggerInfo(cluster.Acme.domains())
	return nil
}
