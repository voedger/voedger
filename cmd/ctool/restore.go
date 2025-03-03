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
	"github.com/voedger/voedger/pkg/coreutils"
)

// nolint
func newRestoreCmd() *cobra.Command {
	restoreCmd := &cobra.Command{
		Use:   "restore <backup name>",
		Short: "Restore the database",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: restore,
	}

	if newCluster().Edition != clusterEditionN1 && !addSshKeyFlag(restoreCmd) {
		return nil
	}

	return restoreCmd
}

func restore(cmd *cobra.Command, args []string) error {

	currentCmd = cmd
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

	if cluster.Edition == clusterEditionN1 {
		if err = resoreCeNode(backupName); err != nil {
			return err
		}
		loggerInfoGreen("CENode restored successfully")
	} else {
		if err = restoreDbNodes(cluster, backupName); err != nil {
			return err
		}
		loggerInfoGreen("DB nodes restored successfully")
	}

	return nil
}

func restoreDbNodes(cluster *clusterType, backupName string) error {

	seConf := newSeConfigType(cluster)

	if err := newScriptExecuter(cluster.sshKey, "").
		run("restore-node.sh", backupName, seConf.DBNode1, seConf.DBNode2, seConf.DBNode3); err != nil {
		return err
	}

	return nil
}

func resoreCeNode(backupName string) error {

	if err := newScriptExecuter("", "").
		run("ce/ce-restore-node.sh", backupName); err != nil {
		return err
	}

	return nil
}

func backupExists(cluster *clusterType, backupPath string) error {

	if cluster.Edition == clusterEditionN1 {
		exists, err := coreutils.Exists(backupPath)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf(errBackupNotExistOnHost, backupPath, "CENode", ErrBackupNotExist)
		}
		return nil
	}

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
