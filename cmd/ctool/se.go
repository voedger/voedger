/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import (
	"errors"
	"fmt"
	"os"
)

func seNodeControllerFunction(n *nodeType) error {

	if len(n.Error) > 0 && (n.DesiredNodeState == nil || n.DesiredNodeState.isEmpty()) {
		n.DesiredNodeState = newNodeState(n.ActualNodeState.Address, n.ActualNodeState.NodeVersion)
	}

	if n.DesiredNodeState == nil || n.DesiredNodeState.isEmpty() {
		return nil
	}

	n.newAttempt()

	var err error

	if err = seNodeValidate(n); err != nil {
		loggerError(err.Error())
		n.Error = err.Error()
		return err
	}

	if err = setHostname(n); err != nil {
		loggerError(err.Error())
		return err
	}

	if err = updateHosts(n); err != nil {
		loggerError(err.Error())
		return err
	}

	if err = deployDocker(n); err != nil {
		loggerError(err.Error())
		return err
	}

	if err = copyCtoolAndKeyToNode(n); err != nil {
		loggerError(err.Error())
		return err
	}

	n.success()
	return nil
}

func setHostname(node *nodeType) error {
	var err error

	loggerInfo(fmt.Sprintf("setting hostname to %s for a %s host...", node.nodeName(), node.DesiredNodeState.Address))

	if err := newScriptExecuter(node.cluster.sshKey, node.DesiredNodeState.Address).
		run("node-set-hostname.sh", node.DesiredNodeState.Address, node.nodeName()); err != nil {
		loggerError(err.Error())
		node.Error = err.Error()
	} else {
		loggerInfo(fmt.Sprintf("set hostname to %s for a %s host with success.", node.nodeName(), node.DesiredNodeState.Address))
	}

	return err
}

// Update hosts file on all nodes in cluster with new value
func updateHosts(node *nodeType) error {
	var err error
	aliveHosts := make(map[string]string)

	for _, clusterNode := range node.cluster.Nodes {
		var ip string
		if clusterNode.ActualNodeState != nil && clusterNode.ActualNodeState.Address != "" {
			ip = clusterNode.ActualNodeState.Address
		} else {
			ip = clusterNode.DesiredNodeState.Address
		}
		aliveHosts[ip] = clusterNode.nodeName()
		if err = newScriptExecuter(node.cluster.sshKey, node.DesiredNodeState.Address).
			run("node-update-hosts.sh", ip, node.DesiredNodeState.Address, node.nodeName()); err != nil {
			loggerError(err.Error())
			node.Error = err.Error()
			break
		} else {
			loggerInfo(fmt.Sprintf("Update /etc/hosts on node %s with values: %s, %s",
				ip,
				node.DesiredNodeState.Address, node.nodeName()))
		}
	}

	if node.cluster.Cmd.Kind == ckReplace {
		for host, hostname := range aliveHosts {
			loggerInfo(fmt.Sprintf("newnode: %s host: %s hostname: %s", node.DesiredNodeState.Address, host, hostname))
			if err = newScriptExecuter(node.cluster.sshKey, node.DesiredNodeState.Address).
				run("node-update-hosts.sh", node.DesiredNodeState.Address, host, hostname); err != nil {
				loggerError(err.Error())
				node.Error = err.Error()
				break
			}
		}
	}

	return err
}

func seNodeValidate(n *nodeType) error {
	loggerInfo(fmt.Sprintf("checking host %s requirements...", n.DesiredNodeState.Address))

	var minRAM string

	if skipNodeMemoryCheck {
		minRAM = "0"
	} else {
		minRAM = n.minAmountOfRAM()
	}

	if err := newScriptExecuter(n.cluster.sshKey, n.DesiredNodeState.Address).
		run("host-validate.sh", n.DesiredNodeState.Address, minRAM); err != nil {
		n.Error = err.Error()
		return err
	}

	loggerInfo(fmt.Sprintf("host %s requirements checked successfully", n.DesiredNodeState.Address))
	return nil
}

func seClusterControllerFunction(c *clusterType) error {

	var err error

	switch c.Cmd.Kind {
	case ckInit, ckUpgrade:
		err = initSeCluster(c)
	case ckReplace:
		var n *nodeType
		if n = c.nodeByHost(c.Cmd.Args[1]); n == nil {
			return fmt.Errorf(errHostNotFoundInCluster, c.Cmd.Args[1], ErrHostNotFoundInCluster)
		}
		switch n.NodeRole {
		case nrDBNode:
			err = replaceSeScyllaNode(c)
		case nrAppNode:
			err = replaceSeAppNode(c)
		}

	default:
		err = ErrUnknownCommand
	}

	if err == nil {
		c.success()
	} else {
		loggerError(err.Error())
	}

	return err
}

func isSkipStack(skipList []string, stack string) bool {
	for _, s := range skipList {
		if s == stack {
			return true
		}
	}
	return false
}

func initSeCluster(cluster *clusterType) error {
	var err error

	if err = deploySeSwarm(cluster); err != nil {
		loggerError(err.Error)
		return err
	}

	if ok := isSkipStack(cluster.Cmd.SkipStacks, "db"); !ok {
		if e := deployDbmsDockerStack(cluster); e != nil {
			loggerError(e.Error)
			err = errors.Join(err, e)
		}
	} else {
		loggerInfo("skipping db stack deployment")
	}

	if ok := isSkipStack(cluster.Cmd.SkipStacks, "app"); !ok {
		if e := deploySeDockerStack(cluster); e != nil {
			loggerError(e.Error)
			err = errors.Join(err, e)
		}
	} else {
		loggerInfo("skipping se stack deployment")
	}

	if ok := isSkipStack(cluster.Cmd.SkipStacks, "mon"); !ok {
		if e := deployMonDockerStack(cluster); e != nil {
			loggerError(e.Error)
			err = errors.Join(err, e)
		}
	} else {
		loggerInfo("skipping mon stack deployment")
	}

	return err
}

func boolToStr(value bool) string {
	result := "0"
	if value {
		result = "1"
	}
	return result
}

func deploySeSwarm(cluster *clusterType) error {

	var err error

	// Init swarm mode
	node := cluster.Nodes[idxSENode1]
	manager := node.nodeName() //ActualNodeState.Address

	err = func() error {

		loggerInfo("swarm init on", manager)
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
	conf := newSeConfigType(cluster)

	for i := 0; i < len(cluster.Nodes); i++ {
		var dc string

		if cluster.Nodes[i].nodeName() == manager {
			continue
		}

		err = func(n *nodeType) error {
			var e error
			loggerInfo("swarm add node on ", n.ActualNodeState.Address)
			if e = newScriptExecuter(cluster.sshKey, node.ActualNodeState.Address).
				run("swarm-add-node.sh", manager, n.nodeName()); e != nil {
				return e
			}

			if e = setNodeSwarmLabels(cluster, n); e != nil {
				return e
			}

			if n.NodeRole == nrDBNode {
				if dc, e = resolveDC(cluster, n.ActualNodeState.Address); e != nil {
					return e
				}

				loggerInfo("Use datacenter: ", dc)

				if e = newScriptExecuter(cluster.sshKey, "localhost").
					run("docker-compose-prepare.sh", conf.DBNode1Name, conf.DBNode2Name, conf.DBNode3Name, boolToStr(devMode)); e != nil {
					return e
				}

				loggerInfo("db node prepare ", n.ActualNodeState.Address)
				if e = newScriptExecuter(cluster.sshKey, n.ActualNodeState.Address).
					run("db-node-prepare.sh", n.nodeName(), dc); e != nil {
					n.Error = e.Error()
					return e
				}
			}
			return nil
		}(&cluster.Nodes[i])
		if err != nil {
			loggerError(err.Error())
			return err
		}

	}

	loggerInfo("swarm deployed successfully")
	return nil
}

func deploySeDockerStack(cluster *clusterType) error {

	loggerInfo("Starting a SE docker stack deployment.")

	conf := newSeConfigType(cluster)

	if err := newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s", conf.AppNode1, conf.AppNode2)).
		run("se-cluster-start.sh", conf.AppNode1Name, conf.AppNode2Name); err != nil {

		return err
	}

	loggerInfo("SE docker stack deployed successfully")
	return nil
}

func deployDbmsDockerStack(cluster *clusterType) error {

	loggerInfo("Starting a DBMS docker stack deployment.")

	conf := newSeConfigType(cluster)

	if err := newScriptExecuter(cluster.sshKey, "localhost").
		run("docker-compose-prepare.sh", conf.DBNode1Name, conf.DBNode2Name, conf.DBNode3Name, boolToStr(devMode)); err != nil {
		loggerError(err.Error())
		return err
	}

	// prepare DBNode1
	loggerInfo("Use datacenter: ", conf.DBNode1DC)
	loggerInfo("db node prepare ", conf.DBNode1)
	if err := newScriptExecuter(cluster.sshKey, conf.DBNode1).
		run("db-node-prepare.sh", conf.DBNode1Name, conf.DBNode1DC); err != nil {
		loggerError(err.Error())
		return err
	}

	// prepare DBNode2
	loggerInfo("use datacenter: ", conf.DBNode2DC)
	loggerInfo("prepare node", conf.DBNode2)
	if err := newScriptExecuter(cluster.sshKey, conf.DBNode2).
		run("db-node-prepare.sh", conf.DBNode2Name, conf.DBNode2DC); err != nil {
		loggerError(err.Error())
		return err
	}

	// prepare DBNode3
	loggerInfo("use datacenter: ", conf.DBNode3DC)
	loggerInfo("prepare node", conf.DBNode3)
	if err := newScriptExecuter(cluster.sshKey, conf.DBNode3).
		run("db-node-prepare.sh", conf.DBNode3Name, conf.DBNode3DC); err != nil {
		loggerError(err.Error())
		return err
	}

	loggerInfo("DBMS docker stack start on", conf.DBNode1, conf.DBNode2, conf.DBNode3)
	if err := newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s %s", conf.DBNode1, conf.DBNode2, conf.DBNode3)).
		run("db-cluster-start.sh", conf.DBNode1Name, conf.DBNode2Name, conf.DBNode3Name); err != nil {
		return err
	}

	loggerInfo("DBMS docker stack deployed successfully")
	return nil
}

// set in swarm all the necessary labels for the cluster node
func setNodeSwarmLabels(cluster *clusterType, node *nodeType) error {

	var err error
	// swarm labels for cluster SE edition
	if cluster.Edition == clusterEditionSE {
		switch node.NodeRole {
		case nrAppNode:
			loggerInfo("swarm set label", node.label(swarmDbmsLabelKey), "on", node.nodeName(), node.address())
			if err = newScriptExecuter(cluster.sshKey, node.address()).
				run("swarm-set-label.sh", node.nodeName(), node.address(), node.label(swarmMonLabelKey), "true"); err != nil {
				return err
			}

			loggerInfo("swarm set label", node.label(swarmAppLabelKey), "on", node.nodeName(), node.address())
			if err = newScriptExecuter(cluster.sshKey, node.address()).
				run("swarm-set-label.sh", node.nodeName(), node.address(), node.label(swarmAppLabelKey), "true"); err != nil {
				return err
			}
		case nrDBNode:
			loggerInfo("swarm set label", node.label(swarmDbmsLabelKey), "on", node.nodeName(), node.address())
			if err = newScriptExecuter(cluster.sshKey, node.ActualNodeState.Address).
				run("swarm-set-label.sh", node.nodeName(), node.ActualNodeState.Address, node.label(swarmDbmsLabelKey), "true"); err != nil {
				return err
			}
		default:
			err = fmt.Errorf(errInvalidNodeRole, node.address(), ErrInvalidNodeRole)
		}
	}

	return err
}

func deployMonDockerStack(cluster *clusterType) error {

	var err error

	loggerInfo("Starting a mon docker stack deployment.")

	conf := newSeConfigType(cluster)

	if err = newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s", conf.AppNode1, conf.AppNode2)).
		run("mon-node-prepare.sh", conf.AppNode1Name, conf.AppNode2Name, conf.DBNode1Name, conf.DBNode2Name, conf.DBNode3Name); err != nil {

		return err
	}

	if err = newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s", conf.AppNode1, conf.AppNode2)).
		run("mon-stack-start.sh", conf.AppNode1Name, conf.AppNode2Name); err != nil {
		return err
	}

	loggerInfo("SE docker mon docker stack deployed successfully")
	return err
}

type seConfigType struct {
	StackName    string
	AppNode1     string
	AppNode2     string
	DBNode1      string
	DBNode2      string
	DBNode3      string
	AppNode1Name string
	AppNode2Name string
	DBNode1Name  string
	DBNode2Name  string
	DBNode3Name  string
	DBNode1DC    string
	DBNode2DC    string
	DBNode3DC    string
}

func newSeConfigType(cluster *clusterType) *seConfigType {

	config := seConfigType{
		StackName: "voedger",
	}

	var err error

	if cluster.Edition == clusterEditionSE {
		config.AppNode1 = cluster.Nodes[idxSENode1].ActualNodeState.Address
		config.AppNode2 = cluster.Nodes[idxSENode2].ActualNodeState.Address
		config.DBNode1 = cluster.Nodes[idxDBNode1].ActualNodeState.Address
		config.DBNode2 = cluster.Nodes[idxDBNode2].ActualNodeState.Address
		config.DBNode3 = cluster.Nodes[idxDBNode3].ActualNodeState.Address
		config.AppNode1Name = cluster.Nodes[idxSENode1].nodeName()
		config.AppNode2Name = cluster.Nodes[idxSENode2].nodeName()
		config.DBNode1Name = cluster.Nodes[idxDBNode1].nodeName()
		config.DBNode2Name = cluster.Nodes[idxDBNode2].nodeName()
		config.DBNode3Name = cluster.Nodes[idxDBNode3].nodeName()
		if config.DBNode1DC, err = resolveDC(cluster, config.DBNode1); err != nil {
			loggerError(err.Error())
			panic(err)
		}
		if config.DBNode2DC, err = resolveDC(cluster, config.DBNode2); err != nil {
			loggerError(err.Error())
			panic(err)
		}
		if config.DBNode3DC, err = resolveDC(cluster, config.DBNode3); err != nil {
			loggerError(err.Error())
			panic(err)
		}
	}

	return &config
}

func deployDocker(node *nodeType) error {
	var err error

	loggerInfo(fmt.Sprintf("deploy docker on a %s host...", node.DesiredNodeState.Address))

	if err = newScriptExecuter(node.cluster.sshKey, node.DesiredNodeState.Address).
		run("docker-install.sh", node.nodeName()); err != nil {
		loggerError(err.Error())
		node.Error = err.Error()
	} else {
		loggerInfo("docker deployed successfully")
	}

	return err
}

func resolveDC(cluster *clusterType, ip string) (dc string, err error) {
	const nodeOffset int32 = 1
	n := cluster.nodeByHost(ip)
	if n == nil {
		return "", fmt.Errorf(errHostNotFoundInCluster, cluster.Cmd.Args[0], ErrHostNotFoundInCluster)
	}
	if (n.idx == int(idxDBNode1+nodeOffset)) || (n.idx == int(idxDBNode2+nodeOffset)) {
		return "dc1", nil
	}
	return "dc2", nil
}

func replaceSeScyllaNode(cluster *clusterType) error {
	var err error
	var dc string

	if dc, err = resolveDC(cluster, cluster.Cmd.Args[1]); err != nil {
		loggerError(err.Error())
		return err
	}
	loggerInfo("Use datacenter: ", dc)

	conf := newSeConfigType(cluster)

	if err = newScriptExecuter(cluster.sshKey, "localhost").
		run("swarm-get-manager-token.sh", conf.AppNode1); err != nil {
		return err
	}

	oldAddr := cluster.Cmd.Args[0]
	newAddr := cluster.Cmd.Args[1]
	if conf.DBNode1 == newAddr {
		conf.DBNode1 = oldAddr
	} else if conf.DBNode2 == newAddr {
		conf.DBNode2 = oldAddr
	} else if conf.DBNode3 == newAddr {
		conf.DBNode3 = oldAddr
	}

	// nolint
	if err = newScriptExecuter(cluster.sshKey, "localhost").
		run("docker-compose-prepare.sh", conf.DBNode1Name, conf.DBNode2Name, conf.DBNode3Name, boolToStr(devMode)); err != nil {
		return err
	}
	// nolint
	if err = newScriptExecuter(cluster.sshKey, "localhost").
		run("db-node-prepare.sh", newAddr, dc); err != nil {
		return err
	}

	// nolint
	if err = newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s, %s", oldAddr, newAddr)).
		run("ctool-scylla-replace-node.sh", oldAddr, newAddr, conf.AppNode1, dc); err != nil {
		return err
	}

	if err = setNodeSwarmLabels(cluster, cluster.nodeByHost(newAddr)); err != nil {
		return err
	}

	loggerInfo(fmt.Sprintf("node %s [%s -> %s] replaced successfully", cluster.nodeByHost(newAddr).nodeName(), oldAddr, newAddr))
	return nil
}

func replaceSeAppNode(cluster *clusterType) error {

	var err error

	conf := newSeConfigType(cluster)

	oldAddr := cluster.Cmd.Args[0]
	newAddr := cluster.Cmd.Args[1]

	var liveOldAddr string

	if conf.AppNode1 == newAddr {
		liveOldAddr = conf.AppNode2
	} else {
		liveOldAddr = conf.AppNode1
	}

	var newNode *nodeType
	if newNode = cluster.nodeByHost(newAddr); newNode == nil {
		return fmt.Errorf(ErrHostNotFoundInCluster.Error(), newAddr)
	}

	// nolint
	if err = newScriptExecuter(cluster.sshKey, "localhost").
		run("swarm-get-manager-token.sh", conf.DBNode1Name); err != nil {
		return err
	}

	loggerInfo("swarm remove node ", oldAddr)
	// nolint
	if err = newScriptExecuter(cluster.sshKey, oldAddr).
		run("swarm-rm-node.sh", conf.DBNode1Name, oldAddr); err != nil {
		return err
	}

	loggerInfo("swarm add node on ", newAddr)
	// nolint
	if err = newScriptExecuter(cluster.sshKey, newAddr).
		run("swarm-add-node.sh", conf.DBNode1Name, newAddr); err != nil {
		return err
	}

	if err = setNodeSwarmLabels(cluster, newNode); err != nil {
		return err
	}

	loggerInfo("copy prometheus data base from", liveOldAddr, "to", newAddr)
	// nolint
	if err = newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s, %s", liveOldAddr, newAddr)).
		run("prometheus-tsdb-copy.sh", liveOldAddr, newAddr); err != nil {
		return err
	}

	loggerInfo("mon node prepare ", newAddr)
	// nolint
	if err = newScriptExecuter(cluster.sshKey, newAddr).
		run("mon-node-prepare.sh", conf.AppNode1Name, conf.AppNode2Name, conf.DBNode1Name, conf.DBNode2Name, conf.DBNode3Name); err != nil {
		return err
	}

	loggerInfo(fmt.Sprintf("node %s [%s -> %s] replaced successfully", newNode.nodeName(), oldAddr, newAddr))
	return nil
}

// check host Available
// pinging the address of the host
func hostIsAvailable(cluster *clusterType, host string) error {
	if err := newScriptExecuter(cluster.sshKey, host).
		run("host-check.sh", host, "only-ping"); err != nil {
		return err
	}
	return nil
}

// node is live
// pinging the address of the node
// checks that the node is alive in the Swarm cluster
func nodeIsLive(node *nodeType) error {
	if err := newScriptExecuter(node.cluster.sshKey, node.nodeName()).
		run("host-check.sh", node.nodeName()); err != nil {
		return err
	}
	return nil
}

func copyCtoolAndKeyToNode(node *nodeType) error {

	ctoolPath, err := os.Executable()

	if err != nil {
		node.Error = err.Error()
		return err
	}

	loggerInfo(fmt.Sprintf("copying ctool and key to %s [%s]", node.nodeName(), node.address()))
	if err := newScriptExecuter(node.cluster.sshKey, node.nodeName()).
		run("copy-ctool.sh", ctoolPath, node.cluster.sshKey, node.address()); err != nil {
		node.Error = err.Error()
		return err
	}

	return nil
}

func setCronBackup(cluster *clusterType, backupTime string) error {

	loggerInfo("Setting a cron schedule for database backup ", backupTime)

	if err := newScriptExecuter(cluster.sshKey, "").
		run("set-cron-backup-ssh.sh", backupTime); err != nil {
		return err
	}

	return nil
}
