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

	loggerInfo(fmt.Sprintf("Setting hostname to %s for a %s host...", node.nodeName(), node.DesiredNodeState.Address))

	if err := newScriptExecuter(node.cluster.sshKey, node.DesiredNodeState.Address).
		run("node-set-hostname.sh", node.DesiredNodeState.Address, node.nodeName()); err != nil {
		loggerError(err.Error())
		node.Error = err.Error()
	} else {
		loggerInfo(fmt.Sprintf("Set hostname to %s for a %s host with success.", node.nodeName(), node.DesiredNodeState.Address))
	}

	return err
}

// Update hosts file on all nodes in cluster with new value
func updateHosts(node *nodeType) error {
	var err error
	var addr string

	hosts := node.cluster.hosts()

	if node.cluster.Cmd.Kind == ckReplace {
		for i := 0; i < len(node.cluster.Nodes); i++ {
			if node.cluster.Nodes[i].DesiredNodeState != nil && node.cluster.Nodes[i].DesiredNodeState.Address != "" {
				addr = node.cluster.Nodes[i].DesiredNodeState.Address
			} else {
				addr = node.cluster.Nodes[i].ActualNodeState.Address
			}
			for hostname, host := range hosts {
				if err = newScriptExecuter(node.cluster.sshKey, node.DesiredNodeState.Address).
					run("node-update-hosts.sh", addr, host, hostname); err != nil {
					loggerError(err.Error())
					node.Error = err.Error()
					break
				}
			}
		}
		return err
	}

	if node.DesiredNodeState != nil && node.DesiredNodeState.Address != "" {
		addr = node.DesiredNodeState.Address
	} else {
		addr = node.ActualNodeState.Address
	}

	for hostname, host := range hosts {

		if err = newScriptExecuter(node.cluster.sshKey, node.DesiredNodeState.Address).
			run("node-update-hosts.sh", addr, host, hostname); err != nil {
			loggerError(err.Error())
			node.Error = err.Error()
			break
		}
	}

	return err
}

func seNodeValidate(n *nodeType) error {
	loggerInfo(fmt.Sprintf("Checking host %s requirements...", n.DesiredNodeState.Address))

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

	loggerInfo(fmt.Sprintf("Host %s requirements checked successfully", n.DesiredNodeState.Address))
	return nil
}

func seClusterControllerFunction(c *clusterType) error {

	var err error

	switch c.Cmd.Kind {
	case ckInit, ckUpgrade:
		err = initSeCluster(c)
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
			err = replaceSeScyllaNode(c)
		case nrAppNode:
			err = replaceSeAppNode(c)
		case nrAppDbNode:
			if err = replaceSeAppNode(c); err == nil {
				err = replaceSeScyllaNode(c)
			}
		}

		if err == nil && len(c.Cron.Backup) > 0 {
			err = setCronBackup(c, c.Cron.Backup)
		}
	case ckAcme:
		if err = deploySeDockerStack(c); err != nil {
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
		loggerInfo("Skipping db stack deployment")
	}

	if ok := isSkipStack(cluster.Cmd.SkipStacks, "app"); !ok {
		if e := deploySeDockerStack(cluster); e != nil {
			loggerError(e.Error)
			err = errors.Join(err, e)
		}
	} else {
		loggerInfo("Skipping app stack deployment")
	}

	if ok := isSkipStack(cluster.Cmd.SkipStacks, "mon"); !ok {
		if e := deployMonDockerStack(cluster); e != nil {
			loggerError(e.Error)
			err = errors.Join(err, e)
		}
	} else {
		loggerInfo("Skipping mon stack deployment")
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
	manager := node.hostNames()[0] //ActualNodeState.Address

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
	conf := newSeConfigType(cluster)

	for i := 0; i < len(cluster.Nodes); i++ {
		var dc string

		if cluster.Nodes[i].hostNames()[0] == manager {
			continue
		}

		err = func(n *nodeType) error {
			var e error
			loggerInfo("Swarm add node on ", n.ActualNodeState.Address)
			if e = newScriptExecuter(cluster.sshKey, node.ActualNodeState.Address).
				run("swarm-add-node.sh", manager, n.hostNames()[0]); e != nil {
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

				loggerInfo("Db node prepare ", n.ActualNodeState.Address)
				if e = newScriptExecuter(cluster.sshKey, n.ActualNodeState.Address).
					run("db-node-prepare.sh", n.hostNames()[0], dc); e != nil {
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

	loggerInfo("Swarm deployed successfully")
	return nil
}

func deploySeDockerStack(cluster *clusterType) error {

	loggerInfo("Starting a app docker stack deployment.")

	conf := newSeConfigType(cluster)

	if err := newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s", conf.AppNode1, conf.AppNode2)).
		run("se-cluster-start.sh", conf.AppNode1Name, conf.AppNode2Name); err != nil {

		return err
	}

	loggerInfo("App docker stack deployed successfully")
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

	s := "Use datacenter:"

	// prepare DBNode1
	loggerInfo(s, conf.DBNode1DC)
	loggerInfo("Db node prepare ", conf.DBNode1)
	if err := newScriptExecuter(cluster.sshKey, conf.DBNode1).
		run("db-node-prepare.sh", conf.DBNode1Name, conf.DBNode1DC); err != nil {
		loggerError(err.Error())
		return err
	}

	// prepare DBNode2
	loggerInfo(s, conf.DBNode2DC)
	loggerInfo("Prepare node", conf.DBNode2)
	if err := newScriptExecuter(cluster.sshKey, conf.DBNode2).
		run("db-node-prepare.sh", conf.DBNode2Name, conf.DBNode2DC); err != nil {
		loggerError(err.Error())
		return err
	}

	// prepare DBNode3
	loggerInfo(s, conf.DBNode3DC)
	loggerInfo("Prepare node", conf.DBNode3)
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
// nolint
func setNodeSwarmLabels(cluster *clusterType, node *nodeType) error {

	var err error
	// swarm labels for cluster N5 (SE) edition
	if cluster.Edition == clusterEditionN5 {
		switch node.NodeRole {
		case nrAppNode:
			loggerInfo("Swarm set label", node.label(swarmDbmsLabelKey), "on", node.nodeName(), node.address())
			if err = newScriptExecuter(cluster.sshKey, node.address()).
				run("swarm-set-label.sh", node.hostNames()[0], node.address(), node.label(swarmMonLabelKey)[0], "true"); err != nil {
				return err
			}

			loggerInfo("Swarm set label", node.label(swarmAppLabelKey), "on", node.nodeName(), node.address())
			if err = newScriptExecuter(cluster.sshKey, node.address()).
				run("swarm-set-label.sh", node.hostNames()[0], node.address(), node.label(swarmAppLabelKey)[0], "true"); err != nil {
				return err
			}
		case nrDBNode:
			loggerInfo("Swarm set label", node.label(swarmDbmsLabelKey), "on", node.nodeName(), node.address())
			if err = newScriptExecuter(cluster.sshKey, node.ActualNodeState.Address).
				run("swarm-set-label.sh", node.hostNames()[0], node.ActualNodeState.Address, node.label(swarmDbmsLabelKey)[0], "true"); err != nil {
				return err
			}
		case nrAppDbNode:

			labels := node.label(swarmMonLabelKey)
			labels = append(labels, node.label(swarmAppLabelKey)...)
			loggerInfo("Swarm set label", node.label(swarmDbmsLabelKey), "on", node.nodeName(), node.address())
			for i := 0; i < len(labels); i++ {
				if err = newScriptExecuter(cluster.sshKey, node.ActualNodeState.Address).
					run("swarm-set-label.sh", node.hostNames()[0], node.ActualNodeState.Address, labels[i], "true"); err != nil {
					return err
				}
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
		//		run("mon-node-prepare.sh", conf.AppNode1Name, conf.AppNode2Name, conf.DBNode1Name, conf.DBNode2Name, conf.DBNode3Name); err != nil {
		run("mon-node-prepare.sh", conf.AppNode1, conf.AppNode2, conf.DBNode1, conf.DBNode2, conf.DBNode3); err != nil {
		return err
	}

	if err = newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s", conf.AppNode1, conf.AppNode2)).
		//		run("mon-stack-start.sh", conf.AppNode1Name, conf.AppNode2Name); err != nil {
		run("mon-stack-start.sh", conf.AppNode1, conf.AppNode2); err != nil {
		return err
	}

	loggerInfo("Adding user voedger to Grafana on app-node-1")
	if err = addGrafanUser(cluster.nodeByHost("app-node-1"), voedger); err != nil {
		return err
	}

	loggerInfo("Adding user voedger to Grafana on app-node-2")
	if err = addGrafanUser(cluster.nodeByHost("app-node-2"), voedger); err != nil {
		return err
	}

	loggerInfo("Voedger's password resetting to monitoring stack")
	if err = setMonPassword(cluster, voedger); err != nil {
		return err
	}

	loggerInfo("Mon docker stack deployed successfully")
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

// nolint
func newSeConfigType(cluster *clusterType) *seConfigType {

	config := seConfigType{
		StackName: "voedger",
	}

	var err error

	if cluster.Edition == clusterEditionN5 {
		config.AppNode1 = cluster.Nodes[idxSENode1].ActualNodeState.Address
		config.AppNode2 = cluster.Nodes[idxSENode2].ActualNodeState.Address
		config.DBNode1 = cluster.Nodes[idxDBNode1].ActualNodeState.Address
		config.DBNode2 = cluster.Nodes[idxDBNode2].ActualNodeState.Address
		config.DBNode3 = cluster.Nodes[idxDBNode3].ActualNodeState.Address
		config.AppNode1Name = cluster.Nodes[idxSENode1].hostNames()[0]
		config.AppNode2Name = cluster.Nodes[idxSENode2].hostNames()[0]
		config.DBNode1Name = cluster.Nodes[idxDBNode1].hostNames()[0]
		config.DBNode2Name = cluster.Nodes[idxDBNode2].hostNames()[0]
		config.DBNode3Name = cluster.Nodes[idxDBNode3].hostNames()[0]

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

	loggerInfo(fmt.Sprintf("Deploy docker on a %s %s host...", node.nodeName(), node.DesiredNodeState.Address))

	if err = newScriptExecuter(node.cluster.sshKey, node.DesiredNodeState.Address).
		run("docker-install.sh", node.DesiredNodeState.Address); err != nil {
		loggerError(err.Error())
		node.Error = err.Error()
	} else {
		loggerInfo(fmt.Sprintf("Docker deployed successfully on a %s %s host", node.nodeName(), node.DesiredNodeState.Address))
	}

	return err
}

func resolveDC(cluster *clusterType, ip string) (dc string, err error) {
	const nodeOffset int32 = 1
	n := cluster.nodeByHost(ip)
	if n == nil {
		return "", fmt.Errorf(errHostNotFoundInCluster, cluster.Cmd.Args[0], ErrHostNotFoundInCluster)
	}

	if cluster.SubEdition == clusterSubEditionSE3 {
		if n.idx < int(idxDBNode1+nodeOffset) {
			return "dc1", nil
		}
		return "dc2", nil
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
	if err = newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s, %s", oldAddr, newAddr)).
		run("ctool-scylla-replace-node.sh", oldAddr, newAddr, conf.AppNode1, dc); err != nil {
		return err
	}

	if err = setNodeSwarmLabels(cluster, cluster.nodeByHost(newAddr)); err != nil {
		return err
	}

	loggerInfo(fmt.Sprintf("db-node %s [%s -> %s] replaced successfully", cluster.nodeByHost(newAddr).nodeName(), oldAddr, newAddr))
	return nil
}

func replaceSeAppNode(cluster *clusterType) error {

	var err error

	conf := newSeConfigType(cluster)

	oldAddr := cluster.Cmd.Args[0]
	newAddr := cluster.Cmd.Args[1]

	var liveOldAddr string
	var liveOldHost string

	if conf.AppNode1 == newAddr {
		liveOldAddr = conf.AppNode2
		liveOldHost = "app-node-2"
	} else {
		liveOldAddr = conf.AppNode1
		liveOldHost = "app-node-1"
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

	loggerInfo("Swarm remove node ", oldAddr)
	// nolint
	if err = newScriptExecuter(cluster.sshKey, oldAddr).
		run("swarm-rm-node.sh", conf.DBNode1Name, oldAddr); err != nil {
		return err
	}

	loggerInfo("Swarm add node on ", newAddr)
	// nolint
	if err = newScriptExecuter(cluster.sshKey, newAddr).
		run("swarm-add-node.sh", conf.DBNode1Name, newAddr); err != nil {
		return err
	}

	if err = setNodeSwarmLabels(cluster, newNode); err != nil {
		return err
	}

	password := "voedger"
	hash, err := hashedPassword(password)
	if err != nil {
		return err
	}

	args := []string{password, hash, liveOldHost}

	if err = newScriptExecuter(cluster.sshKey, "").
		run("prometheus-voedger-password.sh", args...); err != nil {
		return err
	}

	loggerInfo("Copy prometheus data base from", liveOldAddr, "to", newAddr)
	// nolint
	if err = newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s, %s", liveOldAddr, newAddr)).
		run("prometheus-tsdb-copy.sh", liveOldAddr, newAddr); err != nil {
		return err
	}

	loggerInfo("Mon node prepare ", newAddr)
	// nolint
	if err = newScriptExecuter(cluster.sshKey, newAddr).
		run("mon-node-prepare.sh", conf.AppNode1, conf.AppNode2, conf.DBNode1, conf.DBNode2, conf.DBNode3); err != nil {
		return err
	}

	if err = newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s", conf.AppNode1, conf.AppNode2)).
		//		run("mon-stack-start.sh", conf.AppNode1Name, conf.AppNode2Name); err != nil {
		run("mon-stack-start.sh", conf.AppNode1, conf.AppNode2); err != nil {
		return err
	}

	loggerInfo("Adding user voedger to Grafana on ", newNode.nodeName(), newNode.address())
	if err = addGrafanUser(newNode, voedger); err != nil {
		return err
	}

	loggerInfo("Voedger's password resetting to monitoring stack")
	if err = setMonPassword(cluster, voedger); err != nil {
		return err
	}

	loggerInfoGreen(fmt.Sprintf("app-node %s [%s -> %s] replaced successfully", newNode.nodeName(), oldAddr, newAddr))
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

// node is down
// checks that the node is down in the Swarm cluster
func nodeIsDown(node *nodeType) error {
	if node.cluster.SubEdition != clusterSubEditionSE3 {
		if err := newScriptExecuter(node.cluster.sshKey, node.nodeName()).
			run("host-is-down.sh", node.hostNames()[0], node.hostNames()[0]); err != nil {
			return err
		}
	} else {
		if err := newScriptExecuter(node.cluster.sshKey, node.nodeName()).
			run("host-is-down.sh", node.hostNames()[0], node.nodeName()); err != nil {
			return err
		}

	}
	return nil
}

func copyCtoolAndKeyToNode(node *nodeType) error {

	ctoolPath, err := os.Executable()

	if err != nil {
		node.Error = err.Error()
		return err
	}

	loggerInfo(fmt.Sprintf("Copying ctool and key to %s %s", node.nodeName(), node.address()))
	if err := newScriptExecuter(node.cluster.sshKey, node.hostNames()[0]).
		run("copy-ctool.sh", ctoolPath, node.cluster.sshKey, node.address()); err != nil {
		node.Error = err.Error()
		return err
	}

	return nil
}

func setCronBackup(cluster *clusterType, backupTime string) error {

	loggerInfo("Setting a cron schedule for database backup ", backupTime)

	if cluster.Edition == clusterEditionN1 {
		args := []string{backupTime}
		if cluster.Cron.ExpireTime != "" {
			args = append(args, cluster.Cron.ExpireTime)
		}
		if err := newScriptExecuter("", "").
			run("ce/set-cron-backup.sh", args...); err != nil {
			return err
		}
	} else {
		args := []string{backupTime, cluster.SshPort}
		if cluster.Cron.ExpireTime != "" {
			args = append(args, cluster.Cron.ExpireTime)
		}
		if err := newScriptExecuter(cluster.sshKey, "").
			run("set-cron-backup-ssh.sh", args...); err != nil {
			return err
		}
	}

	return nil
}
