/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newBackupCmd() *cobra.Command {
	backupNodeCmd := &cobra.Command{
		Use:   "node <node> <target folder> <path to ssh key>",
		Short: "Backup db node",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 3 {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: backupNode,
	}

	backupCroneCmd := &cobra.Command{
		Use:   "crone <crone event>",
		Short: "Installation of a backup of schedule",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: backupCrone,
	}

	backupCmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup database",
	}

	backupCmd.AddCommand(backupNodeCmd, backupCroneCmd)

	return backupCmd

}

func validateBackupCroneCmd(cmd *cmdType, cluster *clusterType) error {
	return nil
}

func validateBackupNodeCmd(cmd *cmdType, cluster *clusterType) error {
	args := cmd.args()

	if len(args) != 4 {
		return ErrInvalidNumberOfArguments
	}

	var err error

	if n := cluster.nodeByHost(args[1]); n == nil {
		err = errors.Join(err, fmt.Errorf(errHostNotFoundInCluster, args[1], ErrHostNotFoundInCluster))
	}

	if !fileExists(args[3]) {
		err = errors.Join(err, fmt.Errorf(errSshKeyNotFound, args[3], ErrFileNotFound))
	}

	return err
}

func backupNode(cmd *cobra.Command, args []string) error {
	loggerInfo("backup node", args[0])
	cluster := newCluster()

	Cmd := newCmd(ckBackup, "node "+strings.Join(args, " "))

	if err := Cmd.validate(cluster); err != nil {
		return err
	}

	err := mkCommandDirAndLogFile(cmd, cluster)
	if err != nil {
		return err
	}

	loggerInfoGreen("backupnode.sh", strings.Join(args, " "))
	if err = newScriptExecuter("", "").
		run("backupnode.sh", args...); err != nil {
		return err
	}

	return nil
}

func backupCrone(cmd *cobra.Command, args []string) error {
	fmt.Println("backup crone")
	return nil
}
