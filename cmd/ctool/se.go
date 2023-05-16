/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"errors"
	"fmt"

	"github.com/untillpro/goutils/logger"
)

func seNodeControllerFunction(n *nodeType) error {

	if !force && n.DesiredNodeState.isEmpty() {
		return nil
	}

	n.newAttempt()

	var err error

	if err = deployDocker(n); err != nil {
		logger.Error(err.Error())
	} else {
		n.success("ok")
	}

	return err
}

func seClusterControllerFunction(c *clusterType) error {

	var err error

	if err = deploySeSwarm(c); err != nil {
		logger.Error(err.Error)
		return err
	}

	if e := deployDbmsDockerStack(c); e != nil {
		logger.Error(e.Error)
		err = errors.Join(err, e)
	}

	if e := deploySeDockerStack(c); e != nil {
		logger.Error(e.Error)
		err = errors.Join(err, e)
	}

	if e := deployMonDockerStack(c); e != nil {
		logger.Error(e.Error)
		err = errors.Join(err, e)
	}

	if err == nil {
		c.success("ok")
	}

	return err
}

func deploySeSwarm(cluster *clusterType) error {

	var err error

	// Init swarm mode
	node := cluster.Nodes[idxSENode1]
	manager := node.ActualNodeState.Address

	if err = prepareScripts("docker-compose-template.yml", "scylla.yaml", "swarm-init.sh", "swarm-set-label.sh", "db-node-prepare.sh"); err != nil {
		return err
	}

	err = func() error {

		logger.Info("swarm init on", manager)
		if err = newScriptExecuter(cluster.sshKey, manager).
			run("swarm-init.sh", manager); err != nil {
			node.Error = err.Error()
			return err
		}

		logger.Info("swarm set label on", manager, "scylla1")
		if err = newScriptExecuter(cluster.sshKey, manager).
			run("swarm-set-label.sh", manager, manager, node.label()); err != nil {
			node.Error = err.Error()
			return err
		}

		logger.Info("db node prepare", manager)
		if err = newScriptExecuter(cluster.sshKey, manager).
			run("db-node-prepare.sh", manager); err != nil {
			node.Error = err.Error()
			return err
		}
		return nil
	}()

	if err != nil {
		logger.Error(err.Error())
		return err
	}

	// end of Init swarm mode

	// Add remaining nodes to swarm cluster

	if err = prepareScripts("swarm-add-node.sh"); err != nil {
		return err
	}

	for i := 0; i < len(cluster.Nodes); i++ {

		if cluster.Nodes[i].ActualNodeState.Address == manager {
			continue
		}

		func(n *nodeType) {
			logger.Info("swarm add node on ", n.ActualNodeState.Address)
			if e := newScriptExecuter(cluster.sshKey, node.ActualNodeState.Address).
				run("swarm-add-node.sh", manager, n.ActualNodeState.Address); e != nil {
				logger.Error(e.Error())
				n.Error = e.Error()
				return
			}

			logger.Info("swarm set label on", n.ActualNodeState.Address, n.label())
			if e := newScriptExecuter(cluster.sshKey, n.ActualNodeState.Address).
				run("swarm-set-label.sh", manager, n.ActualNodeState.Address, n.label()); e != nil {
				logger.Error(e.Error())
				n.Error = e.Error()
				return
			}

			logger.Info("db node prepare ", n.ActualNodeState.Address)
			if e := newScriptExecuter(cluster.sshKey, n.ActualNodeState.Address).
				run("db-node-prepare.sh", n.ActualNodeState.Address); e != nil {
				logger.Error(e.Error())
				n.Error = e.Error()
				return
			}
		}(&cluster.Nodes[i])
	}

	// end of Add remaining nodes to swarm cluster
	logger.Info("swarm deployed successfully")
	return nil
}

func deploySeDockerStack(cluster *clusterType) error {

	logger.Info("Starting a SE docker stack deployment.")
	if err := prepareScripts("docker-compose-template.yml", "scylla.yaml", "swarm-set-label.sh", "docker-compose-se.yml", "se-cluster-start.sh"); err != nil {
		return err
	}

	//	swarm-set-label вызвать для SENode1 и SENode2

	conf := newSeConfigType(cluster)
	if err := newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s", conf.SENode1, conf.SENode2)).
		run("se-cluster-start.sh", conf.SENode1, conf.SENode2); err != nil {
		return err
	}

	logger.Info("SE docker stack deployed successfully")
	return nil
}

func deployDbmsDockerStack(cluster *clusterType) error {

	logger.Info("Starting a DBMS docker stack deployment.")

	if err := prepareScripts("db-cluster-start.sh"); err != nil {
		return err
	}

	conf := newSeConfigType(cluster)
	logger.Info("db cluster start on", conf.DBNode1, conf.DBNode2, conf.DBNode3)
	if err := newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s %s", conf.DBNode1, conf.DBNode2, conf.DBNode3)).
		run("db-cluster-start.sh", conf.DBNode1, conf.DBNode2, conf.DBNode3); err != nil {
		return err
	}

	logger.Info("DBMS docker stack deployed successfully")
	return nil
}

func deployMonDockerStack(cluster *clusterType) error {

	logger.Info("Starting a mon docker stack deployment.")

	prepareScripts("alertmanager/config.yml", "prometheus/prometheus.yml", "prometheus/alert.rules",
		"docker-compose-mon.yml", "mon-node-prepare.sh", "mon-stack-start.sh", "swarm-set-label.sh")

	conf := newSeConfigType(cluster)

	if err := newScriptExecuter(cluster.sshKey, conf.SENode1).
		run("swarm-set-label.sh", conf.SENode1, conf.SENode1, "mon1"); err != nil {
		return err
	}

	if err := newScriptExecuter(cluster.sshKey, conf.SENode2).
		run("swarm-set-label.sh", conf.SENode1, conf.SENode2, "mon2"); err != nil {
		return err
	}

	if err := newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s", conf.SENode1, conf.SENode2)).
		run("mon-node-prepare.sh", conf.SENode1, conf.SENode2, conf.DBNode1, conf.DBNode2, conf.DBNode3); err != nil {
		return err
	}

	if err := newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s", conf.SENode1, conf.SENode2)).
		run("mon-stack-start.sh", conf.SENode1); err != nil {
		return err
	}

	logger.Info("SE docker mon docker stack deployed successfully")
	return nil
}

type seConfigType struct {
	StackName string
	SENode1   string
	SENode2   string
	DBNode1   string
	DBNode2   string
	DBNode3   string
}

func newSeConfigType(cluster *clusterType) *seConfigType {

	config := seConfigType{
		StackName: "voedger",
	}
	if cluster.Edition == clusterEditionSE {
		config.SENode1 = cluster.Nodes[idxSENode1].ActualNodeState.Address
		config.SENode2 = cluster.Nodes[idxSENode2].ActualNodeState.Address
		config.DBNode1 = cluster.Nodes[idxDBNode1].ActualNodeState.Address
		config.DBNode2 = cluster.Nodes[idxDBNode2].ActualNodeState.Address
		config.DBNode3 = cluster.Nodes[idxDBNode3].ActualNodeState.Address
	}
	return &config
}

func deployDocker(node *nodeType) error {
	var err error

	prepareScripts("docker-install.sh")

	logger.Info(fmt.Sprintf("deploy docker on a %s host...", node.DesiredNodeState.Address))

	if err = newScriptExecuter(node.cluster.sshKey, node.DesiredNodeState.Address).
		run("docker-install.sh", node.DesiredNodeState.Address); err != nil {
		logger.Error(err.Error())
		node.Error = err.Error()
	} else {
		logger.Info("docker deployed successfully")
	}

	return err
}
