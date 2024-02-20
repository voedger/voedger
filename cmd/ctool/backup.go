/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/robfig/cron/v3"
)

var expireTime string

// nolint
func newBackupCmd() *cobra.Command {
	backupNodeCmd := &cobra.Command{
		Use:   "node [<node> <target folder>]",
		Short: "Backup db node",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: backupNode,
	}

	backupNodeCmd.PersistentFlags().StringVarP(&sshPort, "ssh-port", "p", "22", "SSH port")
	backupNodeCmd.PersistentFlags().StringVarP(&expireTime, "expire", "e", "", "Expire time for backup (e.g. 7d, 1m)")
	backupNodeCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to SSH key")
	value, exists := os.LookupEnv(envVoedgerSshKey)
	if !exists || value == "" {
		if err := backupNodeCmd.MarkPersistentFlagRequired("ssh-key"); err != nil {
			loggerError(err.Error())
			return nil
		}
	}

	backupCronCmd := &cobra.Command{
		Use:   "cron [<cron event>]",
		Short: "Installation of a backup of schedule",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: backupCron,
	}
	backupCronCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to SSH key")
	value, exists = os.LookupEnv(envVoedgerSshKey)
	if !exists || value == "" {
		if err := backupCronCmd.MarkPersistentFlagRequired("ssh-key"); err != nil {
			loggerError(err.Error())
			return nil
		}
	}
	backupCronCmd.PersistentFlags().StringVarP(&expireTime, "expire", "e", "", "Expire time for backup (e.g. 7d, 1m)")

	backupListCmd := &cobra.Command{
		Use:   "list",
		Short: "Display a list of existing backups on all DB nodes",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: backupList,
	}
	backupListCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to SSH key")
	if !exists || value == "" {
		if err := backupListCmd.MarkPersistentFlagRequired("ssh-key"); err != nil {
			loggerError(err.Error())
			return nil
		}
	}
	backupCmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup database",
	}

	backupCmd.AddCommand(backupNodeCmd, backupCronCmd, backupListCmd)

	return backupCmd

}

type expireType struct {
	value int
	unit  string
}

func (e *expireType) validate() error {
	if e.unit != "d" && e.unit != "m" {
		return ErrInvalidExpireTime
	}

	if e.value <= 0 {
		return ErrInvalidExpireTime
	}

	return nil
}

func (e *expireType) string() string {
	return fmt.Sprintf("%d%s", e.value, e.unit)
}

func newExpireType(str string) (*expireType, error) {
	unit := string(str[len(str)-1])
	valueStr := str[:len(str)-1]
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return nil, ErrInvalidExpireTime
	}

	expire := &expireType{
		value: value,
		unit:  unit,
	}

	if err := expire.validate(); err != nil {
		return nil, err
	}

	return expire, nil
}

// nolint
func validateBackupCronCmd(cmd *cmdType, cluster *clusterType) error {

	if len(cmd.Args) != 2 {
		return ErrInvalidNumberOfArguments
	}

	if _, err := cron.ParseStandard(cmd.Args[1]); err != nil {
		return err
	}

	return nil
}

// nolint
func validateBackupNodeCmd(cmd *cmdType, cluster *clusterType) error {

	if len(cmd.Args) != 4 {
		return ErrInvalidNumberOfArguments
	}

	var err error

	if n := cluster.nodeByHost(cmd.Args[1]); n == nil {
		err = errors.Join(err, fmt.Errorf(errHostNotFoundInCluster, cmd.Args[1], ErrHostNotFoundInCluster))
	}

	if !fileExists(cmd.Args[3]) {
		err = errors.Join(err, fmt.Errorf(errSshKeyNotFound, cmd.Args[3], ErrFileNotFound))
	}

	return err
}

func backupNode(cmd *cobra.Command, args []string) error {
	cluster := newCluster()

	var err error

	if err = mkCommandDirAndLogFile(cmd, cluster); err != nil {
		return err
	}

	if expireTime != "" {
		expire, e := newExpireType(expireTime)
		if e != nil {
			return e
		}
		cluster.Cron.ExpireTime = expire.string()
	}

	loggerInfo("Backup node", strings.Join(args, " "))
	if err = newScriptExecuter(cluster.sshKey, "").
		run("backup-node.sh", args...); err != nil {
		return err
	}

	cluster.sshKey = args[2]
	if err = deleteExpireBacups(cluster, args[0]); err != nil {
		return err
	}

	return nil
}

func backupCron(cmd *cobra.Command, args []string) error {
	cluster := newCluster()
	if cluster.Draft {
		return ErrClusterConfNotFound
	}

	if expireTime != "" {
		expire, err := newExpireType(expireTime)
		if err != nil {
			return err
		}
		cluster.Cron.ExpireTime = expire.string()
	}

	Cmd := newCmd(ckBackup, append([]string{"cron"}, args...))

	var err error

	if err = Cmd.validate(cluster); err != nil {
		return err
	}

	if err = mkCommandDirAndLogFile(cmd, cluster); err != nil {
		return err
	}

	if err = checkBackupFolders(cluster); err != nil {
		return err
	}

	if err = setCronBackup(cluster, args[0]); err != nil {
		return err
	}

	loggerInfoGreen("Cron schedule set successfully")

	cluster.Cron.Backup = args[0]
	if err = cluster.saveToJSON(); err != nil {
		return err
	}

	return nil
}

// Checking the presence of a Backup folder on DBNodes
func checkBackupFolders(cluster *clusterType) error {
	var err error
	for _, n := range cluster.Nodes {
		if n.NodeRole == nrDBNode {
			if e := newScriptExecuter(cluster.sshKey, "").
				run("check-remote-folder.sh", n.address(), backupFolder); e != nil {
				err = errors.Join(err, fmt.Errorf(errBackupFolderIsNotPrepared, n.nodeName()+" "+n.address(), ErrBackupFolderIsNotPrepared))
			}
		}
	}
	return err
}

// Checking the presence of a Backup folder on node
func checkBackupFolderOnHost(cluster *clusterType, addr string) error {
	if e := newScriptExecuter(cluster.sshKey, "").
		run("check-remote-folder.sh", addr, backupFolder); e != nil {
		return fmt.Errorf(errBackupFolderIsNotPrepared, addr, ErrBackupFolderIsNotPrepared)
	}
	return nil
}

func backupList(cmd *cobra.Command, args []string) error {
	cluster := newCluster()
	if cluster.Draft {
		return ErrClusterConfNotFound
	}

	var err error

	if err = mkCommandDirAndLogFile(cmd, cluster); err != nil {
		return err
	}

	if err = checkBackupFolders(cluster); err != nil {
		return err
	}

	backups, err := getBackupList(cluster)

	loggerInfo(backups)

	return err
}

func getBackupList(cluster *clusterType) (string, error) {

	backupFName := filepath.Join(scriptsTempDir, "backups.lst")

	err := os.Remove(backupFName)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	err = newScriptExecuter(cluster.sshKey, "").run("backup-list.sh")

	fContent, e := ioutil.ReadFile(backupFName)
	if e != nil {
		return "", e
	}

	return string(fContent), err
}

func deleteExpireBacups(cluster *clusterType, hostAddr string) error {

	if cluster.Cron.ExpireTime == "" {
		return nil
	}

	loggerInfo("Search and delete expire backups on", hostAddr)
	if err := newScriptExecuter(cluster.sshKey, "").
		run("delete-expire-backups-ssh.sh", hostAddr, backupFolder, cluster.Cron.ExpireTime); err != nil {
		return err
	}

	return nil
}
