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
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// nolint
func newAlertCmd() *cobra.Command {

	var err error

	alertAddDiscordCmd := &cobra.Command{
		Use:   "discord",
		Short: "Add Discord webhook",
		Args:  cobra.ExactArgs(1),
		RunE:  alertAddDiscord,
	}

	alertRemoveDiscordCmd := &cobra.Command{
		Use:   "discord",
		Short: "Remove Discord webhook",
		Args:  cobra.ExactArgs(0),
		RunE:  alertRemoveDiscord,
	}

	alertAddCmd := &cobra.Command{
		Use:   "add",
		Short: "Add recipients of alerts",
		Args:  cobra.ExactArgs(0),
	}

	alertAddCmd.AddCommand(alertAddDiscordCmd)

	alertRemoveCmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove recipients of alerts",
		Args:  cobra.ExactArgs(0),
	}

	alertRemoveCmd.AddCommand(alertRemoveDiscordCmd)

	alertConfigsCmd := &cobra.Command{
		Use:   "configs",
		Short: "Management of the alert's configuration",
		Args:  cobra.ExactArgs(0),
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

	alertConfigsCmd.AddCommand(alertConfigsDownloadCmd, alertConfigsUploadCmd)

	alertCmd := &cobra.Command{
		Use:   "alert",
		Short: "Management of the alert",
	}

	if err = addSshKeyFlag(alertRemoveDiscordCmd, alertAddDiscordCmd, alertConfigsDownloadCmd, alertConfigsUploadCmd); err != nil {
		return nil
	}

	alertCmd.AddCommand(alertConfigsCmd, alertAddCmd, alertRemoveCmd)

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
		loggerInfo("Removing Discord webhook from", appNode1, "and", appNode2)
	} else {
		loggerInfo("Adding Discord webhook", cluster.Alert.DiscordWebhook, "to", appNode1, "and", appNode2)
	}

	if err = newScriptExecuter(cluster.sshKey, "").
		run("file-upload.sh", localConfigFile, remoteConfigFile, appNode1, appNode2); err != nil {
		return err
	}

	loggerInfo("Restarting alertmanager service on", appNode1, "and", appNode2)
	if err = newScriptExecuter(cluster.sshKey, "").
		run("docker-service-restart.sh", appNode1, "alertmanager"); err != nil {
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
	} else {
		return fmt.Errorf(errIsNotValidUrl, s, ErrIsNotValidUrl)
	}
}

func alertAddDiscord(cmd *cobra.Command, args []string) error {

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

	if err = setDiscordWebhook(cluster, args[0]); err != nil {
		return err
	}

	loggerInfoGreen("Discord webhook added successfully")

	return nil
}

func alertRemoveDiscord(cmd *cobra.Command, args []string) error {

	cluster := newCluster()

	if cluster.Draft {
		return ErrClusterConfNotFound
	}

	var err error

	if err = mkCommandDirAndLogFile(cmd, cluster); err != nil {
		return err
	}

	if err = setDiscordWebhook(cluster, emptyDiscordWebhookUrl); err != nil {
		return err
	}

	loggerInfoGreen("Discord webhook removed successfully")

	return nil
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

	fName := filepath.Join(scriptsTempDir, "post-alert.json")

	if err = os.WriteFile(fName, eventJson, coreutils.FileMode_rwxrwxrwx); err != nil {
		return err
	}

	if err = newScriptExecuter(cluster.sshKey, "").
		run("post-alert.sh", "post-alert.json"); err != nil {
		return err
	}

	return nil
}
