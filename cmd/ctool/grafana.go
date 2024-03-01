/*
* Copyright (c) 2024-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"os"

	"github.com/spf13/cobra"
)

// nolint
func newGrafanaCmd() *cobra.Command {
	grafanaAdminPasswordCmd := &cobra.Command{
		Use:   "admin-password <password>",
		Short: "Set admin password for grafana",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: grafanaAdminPassword,
	}

	grafanaAdminPasswordCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "Path to SSH key")
	value, exists := os.LookupEnv(envVoedgerSshKey)
	if !exists || value == "" {
		if err := grafanaAdminPasswordCmd.MarkPersistentFlagRequired("ssh-key"); err != nil {
			loggerError(err.Error())
			return nil
		}
	}

	grafanaCmd := &cobra.Command{
		Use:   "grafana",
		Short: "Grafana management",
	}

	grafanaCmd.AddCommand(grafanaAdminPasswordCmd)

	return grafanaCmd

}

func checkPassword(password string) error {
	if len(password) < minGrafanaPasswordLength {
		return ErrGrafanaPasswordIsTooShort
	}
	return nil
}

func grafanaAdminPassword(cmd *cobra.Command, args []string) error {
	cluster := newCluster()
	if cluster.Draft {
		return ErrClusterConfNotFound
	}

	password := args[0]
	if err := checkPassword(password); err != nil {
		return err
	}

	if err := newScriptExecuter(cluster.sshKey, "").
		run("grafana-admin-password.sh", args[0]); err != nil {
		return err
	}

	loggerInfoGreen("Password for the admin user in Grafana was successfully changed")
	return nil
}
