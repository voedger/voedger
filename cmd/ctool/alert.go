/*
* Copyright (c) 2024-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */
package main

import (
	"os"
	"path/filepath"

	"github.com/juju/errors"
	"github.com/spf13/cobra"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// nolint
func newAlertCmd() *cobra.Command {
	alertConfigsCmd := &cobra.Command{
		Use:   "configs",
		Short: "Management of the alert's configuration",
	}

	alertConfigsDownloadCmd := &cobra.Command{
		Use:   "download",
		Short: "Download alert's configuration",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: alertConfigsDownload,
	}

	alertConfigsDownloadCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to SSH key")
	value, exists := os.LookupEnv(envVoedgerSshKey)
	if !exists || value == "" {
		if err := alertConfigsDownloadCmd.MarkPersistentFlagRequired("ssh-key"); err != nil {
			loggerError(err.Error())
			return nil
		}
	}

	alertConfigsUploadCmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload alert's configuration",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: alertConfigsUpload,
	}

	alertConfigsUploadCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to SSH key")
	if !exists || value == "" {
		if err := alertConfigsUploadCmd.MarkPersistentFlagRequired("ssh-key"); err != nil {
			loggerError(err.Error())
			return nil
		}
	}
	alertConfigsUploadCmd.PersistentFlags().BoolVar(&jsonFormatBackupList, "json", false, "Output in JSON format")

	alertConfigsCmd.AddCommand(alertConfigsDownloadCmd, alertConfigsUploadCmd)

	alertCmd := &cobra.Command{
		Use:   "alert",
		Short: "Management of the alert",
	}

	alertCmd.AddCommand(alertConfigsCmd)

	return alertCmd

}

func alertConfigsDownload(cmd *cobra.Command, args []string) error {

	cluster := newCluster()

	if cluster.Draft {
		return ErrClusterConfNotFound
	}

	var err error

	if err = mkCommandDirAndLogFile(cmd, cluster); err != nil {
		return err
	}

	host := cluster.nodeByHost("app-node-1").address()
	remoteFile := alertManagerConfigFile
	dir, _ := os.Getwd()
	localFile := filepath.Join(dir, filepath.Base(alertManagerConfigFile))

	if err = newScriptExecuter(cluster.sshKey, "").
		run("file-download.sh", host, remoteFile, localFile); err != nil {
		return err
	}

	loggerInfoGreen("Alert's configuration file downloaded to", localFile)

	return nil
}

func alertConfigsUpload(cmd *cobra.Command, args []string) error {

	cluster := newCluster()

	if cluster.Draft {
		return ErrClusterConfNotFound
	}

	var err error

	if err = mkCommandDirAndLogFile(cmd, cluster); err != nil {
		return err
	}

	remoteFile := alertManagerConfigFile
	dir, _ := os.Getwd()
	localFile := filepath.Join(dir, filepath.Base(alertManagerConfigFile))

	exists, err := coreutils.Exists(localFile)
	if err != nil {
		return err
	}

	if !exists {
		return errors.Errorf("alert's configuration file %s not found", localFile)
	}

	appNode1 := cluster.nodeByHost("app-node-1").address()
	appNode2 := cluster.nodeByHost("app-node-2").address()

	loggerInfo("Uploading alert's configuration file", localFile, "to", appNode1, "and", appNode2)

	if err = newScriptExecuter(cluster.sshKey, "").
		run("file-upload.sh", localFile, remoteFile, appNode1, appNode2); err != nil {
		return err
	}

	loggerInfo("Restarting alertmanager service on", appNode1, "and", appNode2)
	if err = newScriptExecuter(cluster.sshKey, "").
		run("docker-service-restart.sh", appNode1, "alertmanager"); err != nil {
		return err
	}

	loggerInfoGreen("Alert's configuration file uploaded successfully")

	return nil
}
