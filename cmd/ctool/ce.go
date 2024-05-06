/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */
package main

import (
	"fmt"
)

func ceClusterControllerFunction(c *clusterType) error {

	loggerInfo("Deploying monitoring stack...")
	if err := newScriptExecuter(c.sshKey, "").
		run("ce/mon-prepare.sh"); err != nil {
		return err
	}

	loggerInfo("Deploying voedger CE...")
	if err := newScriptExecuter(c.sshKey, "").
		run("ce/ce-start.sh"); err != nil {
		return err
	}

	loggerInfoGreen("CE cluster is deployed successfully.")

	c.success()
	return nil
}

func ceNodeControllerFunction(n *nodeType) error {

	loggerInfo(fmt.Sprintf("Deploying docker on a %s %s host...", n.nodeName(), n.address()))
	if err := newScriptExecuter(n.cluster.sshKey, "").
		run("ce/docker-install.sh"); err != nil {
		return err
	}

	n.success()
	return nil
}

// nolint
func deployCeCluster(cluster *clusterType) error {
	return nil
}
