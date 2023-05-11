/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
)

func seNodeControllerFunction(n *nodeType) error {

	/*
		if n.DesiredNodeState.isEmpty() {
			return nil
		}
	*/

	var err error

	prepareScripts("docker-install.sh")

	logger.Info(fmt.Sprintf("installing docker on a %s node host...", n.ActualNodeState.Address))

	if err = newScriptExecuter(n.cluster.sshKey, n.ActualNodeState.Address).
		run("docker-install.sh", n.ActualNodeState.Address); err != nil {
		logger.Error(err.Error())
		n.Error = err.Error()
	} else {
		n.Info = "ok"
		//n.success("ok")
	}

	return err
}

func seClusterControllerFunction(c *clusterType) error {

	var err error

	if err = deploySwarm(c); err != nil {
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

	if err == nil {
		c.success()
	}

	return err
}

func deploySwarm(cluster *clusterType) error {

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

	logger.Info("Starting a app stack deployment.")
	if err := prepareScripts("docker-compose-template.yml", "scylla.yaml", "swarm-set-label.sh", "docker-compose-se.yml", "se-cluster-start.sh"); err != nil {
		return err
	}

	conf := newSeConfigType(cluster)
	if err := newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s", conf.SENode1, conf.SENode2)).
		run("se-cluster-start.sh", conf.SENode1, conf.SENode2); err != nil {
		return err
	}

	logger.Info("SE docker stack deployed successfully")
	return nil
}

func deployDbmsDockerStack(cluster *clusterType) error {

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
		node.ActualNodeState.NodeVersion = node.desiredNodeVersion(cluster)
	}
	cluster.ActualClusterVersion = cluster.DesiredClusterVersion

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
			logger.Info(fmt.Sprintf("Installing docker on a %sNode host...", n.ActualNodeState.Address))
			if e := newScriptExecuter(cluster.sshKey, n.ActualNodeState.Address).
				run("docker-install.sh", n.ActualNodeState.Address); e != nil {
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
	manager := node.ActualNodeState.Address
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
	return nil
}

func needDeploySeCluster(cluster *clusterType) bool {
	return cluster.DesiredClusterVersion != cluster.ActualClusterVersion
}
