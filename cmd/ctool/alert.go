/*
* Copyright (c) 2024-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */
package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/juju/errors"
	"github.com/spf13/cobra"
	"github.com/voedger/voedger/pkg/coreutils"
)

// nolint
func newAlertCmd() *cobra.Command {

	alertAddDiscordCmd := &cobra.Command{
		Use:   "discord",
		Short: "Add a Discord webhook",
		Args:  cobra.ExactArgs(1),
		RunE:  alertAddDiscord,
	}

	alertRemoveDiscordCmd := &cobra.Command{
		Use:   "discord",
		Short: "Remove a Discord webhook",
		Args:  cobra.ExactArgs(0),
		RunE:  alertRemoveDiscord,
	}

	alertAddCmd := &cobra.Command{
		Use:   "add",
		Short: "Add alert recipients",
		Args:  cobra.ExactArgs(0),
	}

	alertAddCmd.AddCommand(alertAddDiscordCmd)

	alertRemoveCmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove alert recipients",
		Args:  cobra.ExactArgs(0),
	}

	alertRemoveCmd.AddCommand(alertRemoveDiscordCmd)

	alertCmd := &cobra.Command{
		Use:   "alert",
		Short: "Manage alerts",
	}

	if newCluster().Edition != clusterEditionN1 {
		alertConfigsCmd := &cobra.Command{
			Use:   "configs",
			Short: "Manage alert configuration",
			Args:  cobra.ExactArgs(0),
		}

		alertConfigsDownloadCmd := &cobra.Command{
			Use:   "download",
			Short: "Download alert configuration",
			Args: func(cmd *cobra.Command, args []string) error {
				if len(args) != 0 {
					return ErrInvalidNumberOfArguments
				}
				return nil
			},
			RunE: alertConfigsDownload,
		}

		alertConfigsUploadCmd := &cobra.Command{
			Use:   "upload",
			Short: "Upload alert configuration",
			Args: func(cmd *cobra.Command, args []string) error {
				if len(args) != 0 {
					return ErrInvalidNumberOfArguments
				}
				return nil
			},
			RunE: alertConfigsUpload,
		}

		alertConfigsCmd.AddCommand(alertConfigsDownloadCmd, alertConfigsUploadCmd)
		if !addSshKeyFlag(alertConfigsDownloadCmd, alertConfigsUploadCmd, alertRemoveDiscordCmd, alertAddDiscordCmd) {
			return nil
		}
		alertCmd.AddCommand(alertConfigsCmd)
	}

	alertCmd.AddCommand(alertAddCmd, alertRemoveCmd)

	return alertCmd

}

func setDiscordWebhook(cluster *clusterType, webhook string) error {

	var err error

	cluster.Alert.DiscordWebhook = webhook
	localConfigFile := filepath.Join("alertmanager", "config.yml")
	if err = cluster.updateTemplateFile(localConfigFile); err != nil {
		return err
	}

	remoteConfigFile := filepath.Join("~", "alertmanager", "config.yml")
	appNode1 := cluster.nodeByHost("app-node-1").address()
	appNode2 := cluster.nodeByHost("app-node-2").address()

	if webhook == emptyDiscordWebhookUrl {
		loggerInfo(fmt.Sprintf("Removing Discord webhook from %s and %s", appNode1, appNode2))
	} else {
		loggerInfo(fmt.Sprintf("Adding Discord webhook %s to %s and %s", webhook, appNode1, appNode2))
	}

	if err = newScriptExecuter(cluster.sshKey, "").
		run("file-upload.sh", localConfigFile, remoteConfigFile, appNode1, appNode2); err != nil {
		return err
	}

	loggerInfo(fmt.Sprintf("Restarting alertmanager service on %s and %s", appNode1, appNode2))
	if err = newScriptExecuter(cluster.sshKey, "").
		run("docker-service-restart.sh", appNode1, alertmanager); err != nil {
		return err
	}

	if err = cluster.saveToJSON(); err != nil {
		return err
	}

	return nil

}

func setDiscordWebhookCe(cluster *clusterType, webhook string) error {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	alertmanager := "alertmanager"
	configFileName := "config.yml"

	cluster.Alert.DiscordWebhook = webhook
	localConfigFile := filepath.Join("ce", alertmanager, configFileName)
	if err = cluster.updateTemplateFile(localConfigFile); err != nil {
		return err
	}

	remoteDir := filepath.Join(homeDir, alertmanager)
	remoteFile := filepath.Join(remoteDir, configFileName)
	host := n1NodeName

	if webhook == emptyDiscordWebhookUrl {
		loggerInfo("Removing Discord webhook from " + host)
	} else {
		loggerInfo(fmt.Sprintf("Adding Discord webhook %s to %s", webhook, host))
	}

	exists, e := coreutils.Exists(remoteFile)
	if e != nil {
		return e
	}
	if exists {
		if err = os.Remove(remoteFile); err != nil {
			return err
		}
	}

	if err = coreutils.CopyFile(filepath.Join(scriptsTempDir, localConfigFile), remoteDir); err != nil {
		return err
	}

	loggerInfo("Restarting alertmanager on " + host)
	if err = newScriptExecuter(cluster.sshKey, "").
		run("ce/docker-container-restart.sh", alertmanager); err != nil {
		return err
	}

	if err = cluster.saveToJSON(); err != nil {
		return err
	}

	return nil

}

// Checks whether the line is a valid URL
func checkURL(s string) error {
	u, err := url.Parse(s)
	if err == nil && u.Scheme != "" && u.Host != "" {
		return nil
	}

	return fmt.Errorf(errIsNotValidUrl, s, ErrIsNotValidUrl)
}

func alertAddDiscord(cmd *cobra.Command, args []string) error {

	currentCmd = cmd
	cluster := newCluster()

	if cluster.Draft {
		return ErrClusterConfNotFound
	}

	var err error

	if err = mkCommandDirAndLogFile(cmd, cluster); err != nil {
		return err
	}

	if err = checkURL(args[0]); err != nil {
		return err
	}

	if cluster.Edition == clusterEditionN1 {
		if err = setDiscordWebhookCe(cluster, args[0]); err != nil {
			return err
		}
	} else {
		if err = setDiscordWebhook(cluster, args[0]); err != nil {
			return err
		}
	}

	loggerInfoGreen("Discord webhook added successfully")

	return nil
}

func alertRemoveDiscord(cmd *cobra.Command, args []string) error {
	currentCmd = cmd
	cluster := newCluster()

	if cluster.Draft {
		return ErrClusterConfNotFound
	}

	var err error

	if err = mkCommandDirAndLogFile(cmd, cluster); err != nil {
		return err
	}

	if cluster.Edition != clusterEditionN1 {
		if err = setDiscordWebhook(cluster, emptyDiscordWebhookUrl); err != nil {
			return err
		}
	} else {
		if err = setDiscordWebhookCe(cluster, emptyDiscordWebhookUrl); err != nil {
			return err
		}
	}

	loggerInfoGreen("Discord webhook removed successfully")

	return nil
}

func alertConfigsDownload(cmd *cobra.Command, args []string) error {

	currentCmd = cmd
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

	currentCmd = cmd
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

	loggerInfo(fmt.Sprintf("Uploading alert's configuration file %s to %s and %s", localFile, appNode1, appNode2))

	if err = newScriptExecuter(cluster.sshKey, "").
		run("file-upload.sh", localFile, remoteFile, appNode1, appNode2); err != nil {
		return err
	}

	loggerInfo("Restarting alertmanager service on", appNode1, "and", appNode2)
	if err = newScriptExecuter(cluster.sshKey, "").
		run("docker-service-restart.sh", appNode1, alertmanager); err != nil {
		return err
	}

	loggerInfoGreen("Alert's configuration file uploaded successfully")

	return nil
}

type customTime time.Time

func (ct *customTime) MarshalJSON() ([]byte, error) {
	t := time.Time(*ct)
	formatted := t.Format("2006-01-02T15:04:05.99Z")
	return []byte(fmt.Sprintf(`"%s"`, formatted)), nil
}

type eventType struct {
	StartsAt     customTime        `json:"startsAt"`
	EndsAt       customTime        `json:"endsAt"`
	Annotations  map[string]string `json:"annotations"`
	Labels       map[string]string `json:"labels"`
	GeneratorURL string            `json:"generatorURL"`
	Status       string            `json:"status"`
}

func (e *eventType) postAlert(cluster *clusterType) error {

	eventJson, err := json.Marshal([]eventType{*e})
	if err != nil {
		return err
	}

	script := "post-alert.sh"
	dir := scriptsTempDir
	if cluster.Edition == clusterEditionN1 {
		dir = filepath.Join(dir, "ce")
		script = filepath.Join("ce", "post-alert.sh")
	}

	fName := filepath.Join(dir, "post-alert.json")

	if err = os.WriteFile(fName, eventJson, coreutils.FileMode_rwxrwxrwx); err != nil {
		return err
	}

	if err = newScriptExecuter(cluster.sshKey, "").
		run(script, "post-alert.json"); err != nil {
		return err
	}

	return nil
}
