/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

// nolint
func newRestoreCmd() *cobra.Command {
	restoreCmd := &cobra.Command{
		Use:   "restore <backup name>",
		Short: "Backup db node",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: restore,
	}

	restoreCmd.PersistentFlags().StringVarP(&sshPort, "ssh-port", "p", "22", "SSH port")
	restoreCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to SSH key")
	if err := restoreCmd.MarkPersistentFlagRequired("ssh-key"); err != nil {
		loggerError(err.Error())
		return nil
	}
	return restoreCmd
}

func restore(cmd *cobra.Command, args []string) error {

	cluster := newCluster()

	var err error

	if err = mkCommandDirAndLogFile(cmd, cluster); err != nil {
		return err
	}

	backupName := args[0]
	if !filepath.IsAbs(backupName) {
		backupName = filepath.Join(backupFolder, backupName)
	}

	if err = backupExists(cluster, backupName); err != nil {
		return err
	}

	if err = restoreDbNodes(cluster, backupName); err != nil {
		return err
	}

	return nil
}

func restoreDbNodes(cluster *clusterType, backupName string) error {

	seConf := newSeConfigType(cluster)

	if err := newScriptExecuter("", "").
		run("restore-node.sh", backupName, cluster.sshKey, seConf.DBNode1, seConf.DBNode2, seConf.DBNode3); err != nil {
		return err
	}

	return nil
}

func backupExists(cluster *clusterType, backupPath string) error {

	var err error

	for _, node := range cluster.Nodes {
		if node.NodeRole != nrDBNode {
			continue
		}

		if e := newScriptExecuter(cluster.sshKey, "").
			run("check-remote-folder.sh", node.address(), backupPath); e != nil {
			err = errors.Join(err, fmt.Errorf(errBackupNotExistOnHost, backupPath, node.nodeName(), ErrBackupNotExist))
		}
	}

	return err
}
