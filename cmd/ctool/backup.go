/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/voedger/voedger/pkg/coreutils"

	"github.com/robfig/cron/v3"
)

var (
	expireTime           string
	jsonFormatBackupList bool
)

// nolint
func newBackupCmd() *cobra.Command {

	c := newCluster()

	var backupNodeCmd *cobra.Command

	if c.Edition == clusterEditionN1 {
		backupNodeCmd = &cobra.Command{
			Use:   "node [<target folder>]",
			Short: "Backup a database node",
			Args: func(cmd *cobra.Command, args []string) error {
				if len(args) != 1 {
					return ErrInvalidNumberOfArguments
				}
				return nil
			},
			RunE: backupCENode,
		}
	} else {
		backupNodeCmd = &cobra.Command{
			Use:   "node [<node> <target folder>]",
			Short: "Backup a database node",
			Args: func(cmd *cobra.Command, args []string) error {
				if len(args) != 2 {
					return ErrInvalidNumberOfArguments
				}
				return nil
			},
			RunE: backupNode,
		}
		backupNodeCmd.PersistentFlags().StringVarP(&sshPort, "ssh-port", "p", "22", "SSH port")
	}

	backupNodeCmd.PersistentFlags().StringVarP(&expireTime, "expire", "e", "", "Expire time for backup (e.g. 7d, 1m)")

	backupCronCmd := &cobra.Command{
		Use:   "cron [<cron event>]",
		Short: "Install a scheduled backup for the database",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: backupCron,
	}

	backupCronCmd.PersistentFlags().StringVarP(&expireTime, "expire", "e", "", "Expire time for backup (e.g. 7d, 1m)")

	backupListCmd := &cobra.Command{
		Use:   "list",
		Short: "Display a list of existing backups on all database nodes",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: backupList,
	}

	backupListCmd.PersistentFlags().BoolVar(&jsonFormatBackupList, "json", false, "Output in JSON format")

	backupNowCmd := &cobra.Command{
		Use:   "now",
		Short: "Backup the database",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: backupNow,
	}

	backupCmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup the database",
	}

	if c.Edition != clusterEditionN1 && !addSshKeyFlag(backupCmd) {
		return nil
	}
	backupCmd.AddCommand(backupNodeCmd, backupCronCmd, backupListCmd, backupNowCmd)

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

func validateBackupCronCmd(cmd *cmdType, _ *clusterType) error {

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

	exists, errExists := coreutils.Exists(cmd.Args[3])
	if errExists != nil {
		// notest
		err = errors.Join(err, errExists)
		return err
	}
	if !exists {
		err = errors.Join(err, fmt.Errorf(errSshKeyNotFound, cmd.Args[3], ErrFileNotFound))
	}

	return err
}

func newBackupErrorEvent(host string, err error) *eventType {
	return &eventType{
		StartsAt: customTime(time.Now()),
		EndsAt:   customTime(time.Now().Add(time.Minute)),
		Annotations: map[string]string{
			"backup": "Backup failed",
			"error":  err.Error(),
		},
		Labels: map[string]string{
			alertLabelSource:   "ctool",
			alertLabelInstance: host,
			alertLabelSeverity: "error",
		},
		GeneratorURL: fmt.Sprintf("http://%s:9093", host)}
}

func backupCENode(cmd *cobra.Command, args []string) error {

	currentCmd = cmd
	cluster := newCluster()

	var err error

	host := "ce-node"

	if err = mkCommandDirAndLogFile(cmd, cluster); err != nil {
		if e := newBackupErrorEvent(host, err).postAlert(cluster); e != nil {
			err = errors.Join(err, e)
		}
		return err
	}

	if expireTime != "" {
		expire, e := newExpireType(expireTime)
		if e != nil {
			if err := newBackupErrorEvent(host, e).postAlert(cluster); err != nil {
				e = errors.Join(err, e)
			}
			return e
		}
		cluster.Cron.ExpireTime = expire.string()
	}

	loggerInfo("Backup node", strings.Join(args, " "))
	if err = newScriptExecuter("", "").
		run("ce/backup-node.sh", args...); err != nil {
		if e := newBackupErrorEvent(host, err).postAlert(cluster); e != nil {
			err = errors.Join(err, e)
		}
		return err
	}

	if err = deleteExpireBacupsCE(cluster); err != nil {
		return err
	}

	return nil
}

func backupNode(cmd *cobra.Command, args []string) error {

	currentCmd = cmd
	cluster := newCluster()

	var err error

	host := args[0]

	if err = mkCommandDirAndLogFile(cmd, cluster); err != nil {
		if e := newBackupErrorEvent(host, err).postAlert(cluster); e != nil {
			err = errors.Join(err, e)
		}
		return err
	}

	if expireTime != "" {
		expire, e := newExpireType(expireTime)
		if e != nil {
			if err := newBackupErrorEvent(host, e).postAlert(cluster); err != nil {
				e = errors.Join(err, e)
			}
			return e
		}
		cluster.Cron.ExpireTime = expire.string()
	}

	loggerInfo("Backup node", strings.Join(args, " "))
	if err = newScriptExecuter(cluster.sshKey, "").
		run("backup-node.sh", args...); err != nil {
		if e := newBackupErrorEvent(host, err).postAlert(cluster); e != nil {
			err = errors.Join(err, e)
		}
		return err
	}

	if err = deleteExpireBacups(cluster, args[0]); err != nil {
		return err
	}

	return nil
}

func newBackupFolderName() string {
	t := time.Now()
	formattedDate := t.Format("20060102150405")
	return filepath.Join(backupFolder, formattedDate+"-backup")
}

func backupNow(cmd *cobra.Command, args []string) error {
	currentCmd = cmd
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

	folder := newBackupFolderName()

	sBackupNode := "BackupNode"

	for _, n := range cluster.Nodes {
		if n.NodeRole == nrDBNode {
			loggerInfo(sBackupNode, n.nodeName(), n.address())
			if err = newScriptExecuter(cluster.sshKey, "").
				run("backup-node.sh", n.address(), folder); err != nil {
				return err
			}
		}
		if n.NodeRole == nrN1Node {
			loggerInfo(sBackupNode, n.nodeName(), n.address())
			if err = newScriptExecuter("", "").
				run("ce/backup-node.sh", folder); err != nil {
				return err
			}
		}
	}
	return nil
}

func backupCron(cmd *cobra.Command, args []string) error {

	currentCmd = cmd
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
		if n.NodeRole == nrN1Node {
			if e := newScriptExecuter("", "").
				run("ce/check-folder.sh", backupFolder); e != nil {
				err = errors.Join(err, fmt.Errorf(errBackupFolderIsNotPrepared, n.nodeName()+" "+n.address(), ErrBackupFolderIsNotPrepared))
			}
		}
	}
	return err
}

// Checking the presence of a Backup folder on node
// nolint
func checkBackupFolderOnHost(cluster *clusterType, addr string) error {
	if e := newScriptExecuter(cluster.sshKey, "").
		run("check-remote-folder.sh", addr, backupFolder); e != nil {
		return fmt.Errorf(errBackupFolderIsNotPrepared, addr, ErrBackupFolderIsNotPrepared)
	}
	return nil
}

func backupList(cmd *cobra.Command, args []string) error {

	currentCmd = cmd
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

	args := []string{}
	if jsonFormatBackupList {
		args = []string{"json"}
	}

	if cluster.Edition == clusterEditionN1 {
		if err = newScriptExecuter("", "").run("ce/backup-list.sh", args...); err != nil {
			return "", nil
		}
	} else {
		if err = newScriptExecuter(cluster.sshKey, "").run("backup-list.sh", args...); err != nil {
			return "", nil
		}
	}

	fContent, e := os.ReadFile(backupFName)
	if e != nil {
		return "", e
	}

	return string(fContent), nil
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

func deleteExpireBacupsCE(cluster *clusterType) error {

	if cluster.Cron.ExpireTime == "" {
		return nil
	}

	loggerInfo("Search and delete expire backups on ce-node")
	if err := newScriptExecuter("", "").
		run("ce/delete-expire-backups.sh", backupFolder, cluster.Cron.ExpireTime); err != nil {
		return err
	}

	return nil
}
