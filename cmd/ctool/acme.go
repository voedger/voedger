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
		Short: "Add one or more domains to the ACME domain list",
		Args:  cobra.ExactArgs(1),
		RunE:  acmeAdd,
	}

	acmeListCmd := &cobra.Command{
		Use:   "list",
		Short: "Display the list of ACME domains",
		RunE:  acmeList,
	}

	acmeRemoveCmd := &cobra.Command{
		Use:   "remove [<domain1,domain2...>]",
		Short: "Remove one or more domains from the ACME domain list",
		Args:  cobra.ExactArgs(1),
		RunE:  acmeRemove,
	}

	acmeCmd := &cobra.Command{
		Use:   "acme",
		Short: "Manage ACME settings",
	}

	if newCluster().Edition != clusterEditionN1 && !addSshKeyFlag(acmeAddCmd, acmeRemoveCmd) {
		return nil
	}

	acmeCmd.AddCommand(acmeAddCmd, acmeListCmd, acmeRemoveCmd)

	return acmeCmd

}

func acmeAdd(cmd *cobra.Command, args []string) error {
	cluster := newCluster()

	exists, err := cluster.clusterConfigFileExists()
	if err != nil {
		// notest
		return err
	}
	if !exists {
		return ErrClusterConfNotFound
	}

	c := newCmd(ckAcme, append([]string{"add"}, args...))
	if err := cluster.applyCmd(c); err != nil {
		loggerError(err.Error())
		return err
	}

	defer saveClusterToJson(cluster)

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

	exists, err := cluster.clusterConfigFileExists()
	if err != nil {
		// notest
		return err
	}
	if !exists {
		return ErrClusterConfNotFound
	}

	c := newCmd(ckAcme, append([]string{"remove"}, args...))
	if err := cluster.applyCmd(c); err != nil {
		loggerError(err.Error())
		return err
	}

	defer saveClusterToJson(cluster)

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

	exists, err := cluster.clusterConfigFileExists()
	if err != nil {
		// notest
		return err
	}
	if !exists {
		return ErrClusterConfNotFound
	}

	if len(cluster.Acme.domains()) == 0 {
		loggerInfo("ACME domains list is empty")
		return nil
	}

	loggerInfo(cluster.Acme.domains())
	return nil
}
