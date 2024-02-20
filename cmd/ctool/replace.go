/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newReplaceCmd() *cobra.Command {
	replaceCmd := &cobra.Command{
		Use:   "replace",
		Short: "Replaces the cluster node",
		RunE:  replace,
	}
	replaceCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to SSH key")
	value, exists := os.LookupEnv(envVoedgerSshKey)
	if !exists || value == "" {
		if err := replaceCmd.MarkPersistentFlagRequired("ssh-key"); err != nil {
			loggerError(err.Error())
			return nil
		}
	}
	return replaceCmd

}

func replace(cmd *cobra.Command, args []string) error {

	cluster := newCluster()
	var err error

	err = cluster.checkVersion()
	if err != nil {
		return err
	}

	// nolint
	defer cluster.saveToJSON()

	c := newCmd(ckReplace, args)
	if err = cluster.applyCmd(c); err != nil {
		return err
	}

	err = mkCommandDirAndLogFile(cmd, cluster)
	if err != nil {
		return err
	}

	replacedAddress := cluster.Cmd.Args[0]

	if err = cluster.validate(); err == nil {
		if err = cluster.Cmd.apply(cluster); err != nil {
			return err
		}
	}

	fmt.Println("Replaced address:", replacedAddress)
	cluster.ReplacedAddresses = append(cluster.ReplacedAddresses, replacedAddress)
	return nil
}
