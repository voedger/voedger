/*
* Copyright (c) 2024-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
)

func newMonCmd() *cobra.Command {
	monPasswordCmd := &cobra.Command{
		Use:   "password <password>",
		Short: "Set a password for the monitoring stack",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return ErrInvalidNumberOfArguments
			}
			return nil
		},
		RunE: monPassword,
	}

	if newCluster().Edition != clusterEditionN1 && !addSshKeyFlag(monPasswordCmd) {
		return nil
	}

	monCmd := &cobra.Command{
		Use:   "mon",
		Short: "Manage the monitoring stack",
	}

	monCmd.AddCommand(monPasswordCmd)

	return monCmd

}

func monPassword(cmd *cobra.Command, args []string) error {

	currentCmd = cmd
	cluster := newCluster()
	if cluster.Draft {
		return ErrClusterConfNotFound
	}

	password := args[0]

	if err := setMonPassword(cluster, password); err != nil {
		return err
	}

	loggerInfoGreen("Password for the voedger user in monitoring stack was successfully changed")

	return nil
}

func hashedPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func checkMonPassword(password string) error {
	if len(password) < minMonPasswordLength {
		return ErrMonPasswordIsTooShort
	}
	return nil
}

// password installation for voedger user in monitoring stack
func setMonPassword(cluster *clusterType, password string) error {

	var err error

	if err = checkMonPassword(password); err != nil {
		return err
	}

	if err = setGrafanaAdminPassword(cluster, admin); err != nil {
		return err
	}

	if err = setPrometheusPassword(cluster, password); err != nil {
		return err
	}

	scriptName := "g-ds-update.sh"
	if cluster.Edition == clusterEditionN1 {

		if err = newScriptExecuter("", "").
			run(scriptName, cluster.nodeByHost(n1NodeName).address(), admin, admin, password); err != nil {
			return err
		}
	} else {
		if err = newScriptExecuter(cluster.sshKey, "").
			run(scriptName, cluster.nodeByHost("app-node-1").address(), admin, admin, password); err != nil {
			return err
		}

		if err = newScriptExecuter(cluster.sshKey, "").
			run(scriptName, cluster.nodeByHost("app-node-2").address(), admin, admin, password); err != nil {
			return err
		}
	}

	if err = setGrafanaPassword(cluster, password); err != nil {
		return err
	}

	if err = setGrafanaRandomAdminPassword(cluster); err != nil {
		return err
	}

	return nil
}

// installation of a random password for the voedger user in the monitoring stack
// nolint
func setMonRandomPassword(cluster *clusterType) error {

	return setMonPassword(cluster, randomPassword(minMonPasswordLength))
}

// password installation for voedger user in Grafana
func setGrafanaPassword(cluster *clusterType, password string) error {

	var err error

	if err = setGrafanaAdminPassword(cluster, admin); err != nil {
		return err
	}

	if cluster.Edition == clusterEditionN1 {
		if err = newScriptExecuter("", "").
			run("g-user-password-set.sh", cluster.nodeByHost(n1NodeName).address(), admin, admin, password); err != nil {
			return err
		}

	} else {
		if err = newScriptExecuter(cluster.sshKey, "").
			run("g-user-password-set.sh", cluster.nodeByHost("app-node-1").address(), admin, admin, password); err != nil {
			return err
		}

		if err = newScriptExecuter(cluster.sshKey, "").
			run("g-user-password-set.sh", cluster.nodeByHost("app-node-2").address(), admin, admin, password); err != nil {
			return err
		}

		if err = setGrafanaRandomAdminPassword(cluster); err != nil {
			return err
		}
	}

	return nil
}

// password installation for admin user in Grafana
func setGrafanaAdminPassword(cluster *clusterType, password string) error {

	if cluster.Edition == clusterEditionN1 {
		if err := newScriptExecuter("", "").
			run("ce/grafana-admin-password.sh", password); err != nil {
			return err
		}
	} else {

		if err := newScriptExecuter(cluster.sshKey, "").
			run("grafana-admin-password.sh", password, cluster.nodeByHost("app-node-1").address()); err != nil {
			return err
		}

		if err := newScriptExecuter(cluster.sshKey, "").
			run("grafana-admin-password.sh", password, cluster.nodeByHost("app-node-2").address()); err != nil {
			return err
		}
	}

	return nil
}

// random password installation for admin user in Grafana
func setGrafanaRandomAdminPassword(cluster *clusterType) error {

	if err := setGrafanaAdminPassword(cluster, randomPassword(minMonPasswordLength)); err != nil {
		return err
	}

	return nil
}

// password installation for voedger user in Prometheus
func setPrometheusPassword(cluster *clusterType, password string) error {

	hash, err := hashedPassword(password)
	if err != nil {
		return err
	}

	if cluster.Edition == clusterEditionN1 {
		args := []string{password, hash}

		if err = newScriptExecuter("", "").
			run("ce/prometheus-voedger-password.sh", args...); err != nil {
			return err
		}

	} else {
		args := append([]string{password, hash}, "app-node-1", "app-node-2")

		if err = newScriptExecuter(cluster.sshKey, "").
			run("prometheus-voedger-password.sh", args...); err != nil {
			return err
		}
	}

	return nil
}

// rendom password installation for voedger user in Prometheus
// nolint
func setPrometheusRandomPassword(cluster *clusterType) error {

	return setPrometheusPassword(cluster, randomPassword(minMonPasswordLength))
}

// adding to Grafana user voedger
func addGrafanUser(node *nodeType, password string) error {

	var err error

	if err = checkMonPassword(password); err != nil {
		return err
	}

	if err = setGrafanaAdminPassword(node.cluster, admin); err != nil {
		return err
	}

	if err = newScriptExecuter(node.cluster.sshKey, "").
		run("g-user-preferences-set.sh", node.address(), admin, admin); err != nil {
		return err
	}

	if err = newScriptExecuter(node.cluster.sshKey, "").
		run("g-user-add.sh", node.address(), admin, admin); err != nil {
		return err
	}

	if err = newScriptExecuter(node.cluster.sshKey, "").
		run("g-ds-update.sh", node.address(), admin, admin, password); err != nil {
		return err
	}

	return nil

}
