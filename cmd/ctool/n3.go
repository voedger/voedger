/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"errors"
	"fmt"
)

// N3 cluster controller function - handles 3-node cluster deployment
func n3ClusterControllerFunction(c *clusterType) error {
	var err error

	switch c.Cmd.Kind {
	case ckInit, ckUpgrade:
		err = initN3Cluster(c)
		if err == nil && len(c.Cron.Backup) > 0 {
			err = setCronBackup(c, c.Cron.Backup)
		}
	case ckReplace:
		var n *nodeType
		if n = c.nodeByHost(c.Cmd.Args[1]); n == nil {
			return fmt.Errorf(errHostNotFoundInCluster, c.Cmd.Args[1], ErrHostNotFoundInCluster)
		}
		switch n.NodeRole {
		case nrDBNode:
			err = replaceN3ScyllaNode(c)
		case nrAppDbNode:
			if err = replaceN3AppNode(c); err == nil {
				err = replaceN3ScyllaNode(c)
			}
		}

		if err == nil && len(c.Cron.Backup) > 0 {
			err = setCronBackup(c, c.Cron.Backup)
		}
	case ckAcme:
		if err = deployN3DockerStack(c); err != nil {
			return err
		}
	default:
		err = ErrUnknownCommand
	}

	if err == nil {
		c.success()
	}

	return err
}

// Initialize N3 cluster - 3 nodes with combined app/db roles
func initN3Cluster(cluster *clusterType) error {
	var err error

	if err = deployN3Swarm(cluster); err != nil {
		loggerError(err.Error)
		return err
	}

	if ok := isSkipStack(cluster.Cmd.SkipStacks, "db"); !ok {
		if e := deployN3DbmsDockerStack(cluster); e != nil {
			loggerError(e.Error)
			err = errors.Join(err, e)
		}
	} else {
		loggerInfo("Skipping db stack deployment")
	}

	if ok := isSkipStack(cluster.Cmd.SkipStacks, "app"); !ok {
		if e := deployN3DockerStack(cluster); e != nil {
			loggerError(e.Error)
			err = errors.Join(err, e)
		}
	} else {
		loggerInfo("Skipping app stack deployment")
	}

	if ok := isSkipStack(cluster.Cmd.SkipStacks, "mon"); !ok {
		if e := deployN3MonStack(cluster); e != nil {
			loggerError(e.Error)
			err = errors.Join(err, e)
		}
	} else {
		loggerInfo("Skipping mon stack deployment")
	}

	if err == nil {
		loggerInfo("N3 cluster deployed successfully")
	}

	return err
}

// Deploy Docker Swarm for N3 cluster
func deployN3Swarm(cluster *clusterType) error {
	var err error

	// Init swarm mode on first node
	node := cluster.Nodes[0] // First node becomes manager
	manager := node.hostNames()[0]

	err = func() error {
		loggerInfo("Swarm init on", manager)
		if err = newScriptExecuter(cluster.sshKey, manager).
			run("swarm-init.sh", manager); err != nil {
			node.Error = err.Error()
			return err
		}

		if err = setNodeSwarmLabels(cluster, &node); err != nil {
			node.Error = err.Error()
			return err
		}

		return nil
	}()

	if err != nil {
		loggerError(err.Error())
		return err
	}

	// Add remaining nodes to swarm cluster
	for i := 0; i < len(cluster.Nodes); i++ {
		var dc string

		if cluster.Nodes[i].hostNames()[0] == manager {
			continue
		}

		switch cluster.Nodes[i].NodeRole {
		case nrAppDbNode:
			dc = "dc1"
		default:
			dc = "dc1"
		}

		loggerInfo("Swarm join", cluster.Nodes[i].hostNames()[0])
		if err = newScriptExecuter(cluster.sshKey, cluster.Nodes[i].hostNames()[0]).
			run("swarm-join.sh", manager, dc); err != nil {
			cluster.Nodes[i].Error = err.Error()
			return err
		}

		if err = setNodeSwarmLabels(cluster, &cluster.Nodes[i]); err != nil {
			cluster.Nodes[i].Error = err.Error()
			return err
		}
	}

	loggerInfo("Docker swarm deployed successfully")
	return nil
}

// Deploy DBMS stack for N3 cluster
func deployN3DbmsDockerStack(cluster *clusterType) error {
	conf := newN3ConfigType(cluster)

	loggerInfo("DBMS docker stack start on", conf.DBNode1, conf.DBNode2, conf.DBNode3)
	if err := newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s %s", conf.DBNode1, conf.DBNode2, conf.DBNode3)).
		run("db-cluster-start.sh", conf.DBNode1Name, conf.DBNode2Name, conf.DBNode3Name); err != nil {
		return err
	}

	loggerInfo("DBMS docker stack deployed successfully")
	return nil
}

// Deploy application stack for N3 cluster
func deployN3DockerStack(cluster *clusterType) error {
	conf := newN3ConfigType(cluster)

	loggerInfo("App docker stack start on", conf.AppNode1, conf.AppNode2)
	if err := newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s", conf.AppNode1, conf.AppNode2)).
		run("app-cluster-start.sh", conf.AppNode1Name, conf.AppNode2Name); err != nil {
		return err
	}

	loggerInfo("App docker stack deployed successfully")
	return nil
}

// Deploy monitoring stack for N3 cluster
func deployN3MonStack(cluster *clusterType) error {
	conf := newN3ConfigType(cluster)

	loggerInfo("Mon docker stack start on", conf.AppNode1)
	if err := newScriptExecuter(cluster.sshKey, conf.AppNode1).
		run("mon-cluster-start.sh", conf.AppNode1Name); err != nil {
		return err
	}

	loggerInfo("Mon docker stack deployed successfully")
	return nil
}

// Replace application node in N3 cluster
func replaceN3AppNode(cluster *clusterType) error {
	// Implementation for replacing app node in N3 cluster
	return fmt.Errorf("replace N3 app node not yet implemented")
}

// Replace database node in N3 cluster
func replaceN3ScyllaNode(cluster *clusterType) error {
	// Implementation for replacing DB node in N3 cluster
	return fmt.Errorf("replace N3 scylla node not yet implemented")
}

// N3 configuration type
type n3ConfigType struct {
	AppNode1     string
	AppNode1Name string
	AppNode2     string
	AppNode2Name string
	DBNode1      string
	DBNode1Name  string
	DBNode2      string
	DBNode2Name  string
	DBNode3      string
	DBNode3Name  string
}

// Create N3 configuration from cluster
func newN3ConfigType(cluster *clusterType) *n3ConfigType {
	conf := &n3ConfigType{}

	// N3 cluster has 3 nodes:
	// Node 1: App + DB (app-node-1, db-node-1)
	// Node 2: App + DB (app-node-2, db-node-2)
	// Node 3: DB only (db-node-3)

	conf.AppNode1 = cluster.Nodes[0].address()
	conf.AppNode1Name = "app-node-1"
	conf.DBNode1 = cluster.Nodes[0].address()
	conf.DBNode1Name = "db-node-1"

	conf.AppNode2 = cluster.Nodes[1].address()
	conf.AppNode2Name = "app-node-2"
	conf.DBNode2 = cluster.Nodes[1].address()
	conf.DBNode2Name = "db-node-2"

	conf.DBNode3 = cluster.Nodes[2].address()
	conf.DBNode3Name = "db-node-3"

	return conf
}
