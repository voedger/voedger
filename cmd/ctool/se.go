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
		config.SENode1 = cluster.Nodes[idxSENode1].State.Address
		config.SENode2 = cluster.Nodes[idxSENode2].State.Address
		config.DBNode1 = cluster.Nodes[idxDBNode1].State.Address
		config.DBNode2 = cluster.Nodes[idxDBNode2].State.Address
		config.DBNode3 = cluster.Nodes[idxDBNode3].State.Address
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
			logger.Info(fmt.Sprintf("Installing docker on a %s host...", n.State.Address))
			if e := newScriptExecuter(cluster.sshKey, n.State.Address).
				run("docker-install.sh", n.State.Address); e != nil {
				logger.Error(e.Error())
				n.State.Error = e.Error()
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
	manager := node.State.Address
	if err = prepareScripts("swarm-init.sh", "swarm-set-label.sh", "db-node-prepare.sh"); err != nil {
		return err
	}

	err = func() error {

		logger.Info("swarm init on", manager)
		if err = newScriptExecuter(cluster.sshKey, manager).
			run("swarm-init.sh", manager); err != nil {
			node.State.Error = err.Error()
			return err
		}

		logger.Info("swarm set label on", manager, "scylla1")
		if err = newScriptExecuter(cluster.sshKey, manager).
			run("swarm-set-label.sh", manager, manager, node.label()); err != nil {
			node.State.Error = err.Error()
			return err
		}

		logger.Info("db node prepare", manager)
		if err = newScriptExecuter(cluster.sshKey, manager).
			run("db-node-prepare.sh", manager); err != nil {
			node.State.Error = err.Error()
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

		if cluster.Nodes[i].State.Address == manager {
			continue
		}

		func(n *nodeType) {
			logger.Info("swarm add node on ", n.State.Address)
			if e := newScriptExecuter(cluster.sshKey, node.State.Address).
				run("swarm-add-node.sh", manager, n.State.Address); e != nil {
				logger.Error(e.Error())
				n.State.Error = e.Error()
				return
			}

			logger.Info("swarm set label on", n.State.Address, n.label())
			if e := newScriptExecuter(cluster.sshKey, n.State.Address).
				run("swarm-set-label.sh", manager, n.State.Address, n.label()); e != nil {
				logger.Error(e.Error())
				n.State.Error = e.Error()
				return
			}

			logger.Info("db node prepare ", n.State.Address)
			if e := newScriptExecuter(cluster.sshKey, n.State.Address).
				run("db-node-prepare.sh", n.State.Address); e != nil {
				logger.Error(e.Error())
				n.State.Error = e.Error()
				return
			}
		}(&cluster.Nodes[i])
	}

	// end of Add remaining nodes to swarm cluster

	if nodes, err = cluster.nodesForProcess(); len(nodes) == 0 {
		return err
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

	if nodes, err = cluster.nodesForProcess(); len(nodes) == 0 {
		return err
	}

	// succses!
	for _, node := range nodes {
		node.State.NodeVersion = cluster.CToolVersion
	}
	logger.Info("cluster successfully deployed!")
	return nil

}
