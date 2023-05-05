/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
)

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
		config.SENode1 = cluster.Nodes[idxSENode1].ActualState.Address
		config.SENode2 = cluster.Nodes[idxSENode2].ActualState.Address
		config.DBNode1 = cluster.Nodes[idxDBNode1].ActualState.Address
		config.DBNode2 = cluster.Nodes[idxDBNode2].ActualState.Address
		config.DBNode3 = cluster.Nodes[idxDBNode3].ActualState.Address
	}
	return &config
}

// temporary verification
// in the future it will be added to the scripts
func checkSwarmStatus(host string) bool {
	s := fmt.Sprintf("docker node inspect --format {{.Status.State}} ip-%s", strings.ReplaceAll(host, ".", "-"))
	sout, _, _ := new(exec.PipedExec).
		Command("bash", "-c", s).
		RunToStrings()

	return string(sout) == "ready"
}

func deploySeCluster(cluster *clusterType) error {

	defer cluster.saveToJSON()
	logger.Info("Starting a cluster deployment.")

	if !cluster.needStartProcess() {
		logger.Info("The cluster has already been successfully deployed.")
		return nil
	}

	var err error

	if needPrepareSeNodes(cluster) {
		if err = prepareSeNodes(cluster); err != nil {
			logger.Error(err.Error())
			return err
		}
	}

	// Start db cluster

	prepareScripts("db-cluster-start.sh")

	conf := newSeConfigType(cluster)

	logger.Info("db cluster start on", conf.DBNode1, conf.DBNode2, conf.DBNode3)
	if err = newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s %s", conf.DBNode1, conf.DBNode2, conf.DBNode3)).
		run("db-cluster-start.sh", conf.DBNode1, conf.DBNode2, conf.DBNode3); err != nil {
		logger.Error(err.Error())
		return err
	}

	// end of Start db cluster
	logger.Info(">>>>>  deploy Se App Stack", conf.SENode1, conf.SENode2)
	if err = deploySeDockerStack(cluster); err != nil {
		return err
	}

	if nodes, err := cluster.nodesForProcess(); len(nodes) == 0 {
		return err
	}

	// succses!
	for _, node := range cluster.Nodes {
		node.ActualState.NodeVersion = node.desiredNodeVersion(cluster)
	}
	cluster.ActualVersion = cluster.DesiredVersion

	logger.Info("cluster successfully deployed!")
	return nil

}

func needPrepareSeNodes(cluster *clusterType) bool {
	for _, n := range cluster.Nodes {
		if err := n.check(cluster); err != nil {
			return true
		}
	}
	return false
}

func prepareSeNodes(cluster *clusterType) error {

	nodes, err := cluster.nodesForProcess()
	if err != nil {
		return err
	}

	// Install yq
	prepareScripts("yq-install.sh")

	if err := newScriptExecuter("", "localhost").run("yq-install.sh"); err != nil {
		logger.Error(err.Error())
		return err
	}
	// end of Install yq

	// Install docker

	prepareScripts("docker-install.sh")

	var wg sync.WaitGroup

	for i := 0; i < len(nodes); i++ {
		wg.Add(1)

		go func(n *nodeType) {
			defer wg.Done()
			logger.Info(fmt.Sprintf("Installing docker on a %s host...", n.ActualState.Address))
			if e := newScriptExecuter(cluster.sshKey, n.ActualState.Address).
				run("docker-install.sh", n.ActualState.Address); e != nil {
				logger.Error(e.Error())
				n.Error = e.Error()
			}
		}(nodes[i])
	}

	wg.Wait()

	// end of Install docker

	if nodes, err = cluster.nodesForProcess(); len(nodes) == 0 {
		return err
	}

	prepareScripts("docker-compose-template.yml", "scylla.yaml")

	// Init swarm mode
	node := cluster.Nodes[idxSENode1]
	manager := node.ActualState.Address
	if err = prepareScripts("swarm-init.sh", "swarm-set-label.sh", "db-node-prepare.sh"); err != nil {
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
	}
	// end of Init swarm mode

	// Add remaining nodes to swarm cluster

	if err = prepareScripts("swarm-add-node.sh"); err != nil {
		return err
	}

	for i := 0; i < len(cluster.Nodes); i++ {

		if cluster.Nodes[i].ActualState.Address == manager {
			continue
		}

		func(n *nodeType) {
			logger.Info("swarm add node on ", n.ActualState.Address)
			if e := newScriptExecuter(cluster.sshKey, node.ActualState.Address).
				run("swarm-add-node.sh", manager, n.ActualState.Address); e != nil {
				logger.Error(e.Error())
				n.Error = e.Error()
				return
			}

			logger.Info("swarm set label on", n.ActualState.Address, n.label())
			if e := newScriptExecuter(cluster.sshKey, n.ActualState.Address).
				run("swarm-set-label.sh", manager, n.ActualState.Address, n.label()); e != nil {
				logger.Error(e.Error())
				n.Error = e.Error()
				return
			}

			logger.Info("db node prepare ", n.ActualState.Address)
			if e := newScriptExecuter(cluster.sshKey, n.ActualState.Address).
				run("db-node-prepare.sh", n.ActualState.Address); e != nil {
				logger.Error(e.Error())
				n.Error = e.Error()
				return
			}
		}(&cluster.Nodes[i])
	}

	// end of Add remaining nodes to swarm cluster
	return nil
}

func needDeploySeCluster(cluster *clusterType) bool {
	return cluster.DesiredVersion != cluster.ActualVersion
}

func deploySeDockerStack(cluster *clusterType) error {

	logger.Info("Starting a app stack deployment.")
	prepareScripts("swarm-set-label.sh", "docker-compose-se.yml", "se-cluster-start.sh")

	conf := newSeConfigType(cluster)
	if err := newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s", conf.SENode1, conf.SENode2)).
		run("se-cluster-start.sh", conf.SENode1, conf.SENode2); err != nil {
		logger.Error(err.Error())
		return err
	}

	return nil
}

func deployDbmsDockerStack(cluster *clusterType) error {

	prepareScripts("db-cluster-start.sh")

	conf := newSeConfigType(cluster)
	logger.Info("db cluster start on", conf.DBNode1, conf.DBNode2, conf.DBNode3)
	if err := newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s %s", conf.DBNode1, conf.DBNode2, conf.DBNode3)).
		run("db-cluster-start.sh", conf.DBNode1, conf.DBNode2, conf.DBNode3); err != nil {
		logger.Error(err.Error())
		return err
	}

	return nil
}
