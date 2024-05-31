/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */
package main

import (
	"fmt"
	"os"
	"path/filepath"

	coreutils "github.com/voedger/voedger/pkg/utils"
)

func ceClusterControllerFunction(c *clusterType) error {

	var err error

	switch c.Cmd.Kind {
	case ckInit, ckUpgrade, ckAcme:
		loggerInfo("Deploying monitoring stack...")
		if err = newScriptExecuter("", "").
			run("ce/mon-prepare.sh"); err != nil {
			return err
		}

		loggerInfo("Deploying voedger CE...")
		if err = newScriptExecuter("", "").
			run("ce/ce-start.sh"); err != nil {
			return err
		}

		loggerInfo("Adding user voedger to Grafana on ce-node")
		if err = addGrafanUser(c.nodeByHost(ceNodeName), voedger); err != nil {
			return err
		}

		loggerInfo("Voedger's password resetting to monitoring stack")
		if err = setMonPassword(c, voedger); err != nil {
			return err
		}

	default:
		err = ErrUnknownCommand
	}

	if err == nil {
		loggerInfoGreen("CE cluster is deployed successfully.")

		c.success()
	}

	return err
}

func ceNodeControllerFunction(n *nodeType) error {

	loggerInfo(fmt.Sprintf("Deploying docker on a %s %s host...", n.nodeName(), n.address()))
	if err := newScriptExecuter(n.cluster.sshKey, "").
		run("ce/docker-install.sh"); err != nil {
		return err
	}

	if err := copyCtoolToCeNode(n); err != nil {
		return err
	}

	n.success()
	return nil
}

// nolint
func deployCeCluster(cluster *clusterType) error {
	return nil
}

func copyCtoolToCeNode(node *nodeType) error {

	ctoolPath, err := os.Executable()

	ok, e := coreutils.Exists(node.cluster.configFileName)

	if e != nil {
		return e
	}

	if !ok {
		if e := node.cluster.saveToJSON(); err != nil {
			return e
		}
	}

	if err != nil {
		node.Error = err.Error()
		return err
	}

	loggerInfo(fmt.Sprintf("Copying ctool and configuration file to %s", ctoolPath))
	if err := newScriptExecuter("", "").
		run("ce/copy-ctool.sh", filepath.Dir(ctoolPath)); err != nil {
		node.Error = err.Error()
		return err
	}

	return nil
}
