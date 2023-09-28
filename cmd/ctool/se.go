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

	if len(n.Error) > 0 && (n.DesiredNodeState == nil || n.DesiredNodeState.isEmpty()) {
		n.DesiredNodeState = newNodeState(n.ActualNodeState.Address, n.ActualNodeState.NodeVersion)
	}

	if n.DesiredNodeState == nil || n.DesiredNodeState.isEmpty() {
		return nil
	}

	n.newAttempt()

	var err error

	if err = seNodeValidate(n); err != nil {
		logger.Error(err.Error())
		return err
	}

	if err = setHostname(n); err != nil {
		logger.Error(err.Error())
		return err
	}

	/*	if n.cluster.Cmd.Kind == ckReplace {
		cmd := fmt.Sprintf("ssh-keygen -f \"~/.ssh/known_hosts\" -R \"%s\"", n.nodeName())
		parts := strings.Fields(cmd)
		var output []byte
		command := parts[0]
		args := parts[1:]
		// Run the command
		output, err = exec.Command(command, args...).Output()
		if err != nil {
			logger.Error(err.Error())
			return err
		}
		logger.Info(string(output))
	}*/

	if err = updateHosts(n); err != nil {
		logger.Error(err.Error())
		return err
	}

	if err = deployDocker(n); err != nil {
		logger.Error(err.Error())
	} else {
		n.success()
	}

	return err
}

func setHostname(node *nodeType) error {
	var err error

	prepareScripts("node-set-hostname.sh")

	logger.Info(fmt.Sprintf("settting hostname to %s for a %s host...", node.nodeName(), node.DesiredNodeState.Address))

	if err = newScriptExecuter(node.cluster.sshKey, node.DesiredNodeState.Address).
		run("node-set-hostname.sh", node.DesiredNodeState.Address, node.nodeName()); err != nil {
		logger.Error(err.Error())
		node.Error = err.Error()
	} else {
		logger.Info(fmt.Sprintf("set hostname to %s for a %s host with success.", node.nodeName(), node.DesiredNodeState.Address))
	}

	return err
}

// Update hosts file on all nodes in cluster with new value
func updateHosts(node *nodeType) error {
	var err error
	aliveHosts := make(map[string]string)
	if err = prepareScripts("node-update-hosts.sh"); err != nil {
		logger.Error(err.Error())
		node.Error = err.Error()
		return err
	}

	for _, clusterNode := range node.cluster.Nodes {
		var ip string
		if clusterNode.ActualNodeState == nil {
			ip = clusterNode.DesiredNodeState.Address
		} else {
			ip = clusterNode.ActualNodeState.Address
		}
		/*		if clusterNode.ActualNodeState.Address != "" {
					ip = clusterNode.ActualNodeState.Address
				} else {
					ip = clusterNode.DesiredNodeState.Address
					newNode = ip
				}*/
		aliveHosts[ip] = clusterNode.nodeName()
		if err = newScriptExecuter(node.cluster.sshKey, node.DesiredNodeState.Address).
			run("node-update-hosts.sh", ip, node.DesiredNodeState.Address, node.nodeName()); err != nil {
			logger.Error(err.Error())
			node.Error = err.Error()
			break
		} else {
			logger.Info(fmt.Sprintf("Update /etc/hosts on node %s with values: %s, %s",
				ip,
				node.DesiredNodeState.Address, node.nodeName()))
		}
	}

	for host, hostname := range aliveHosts {
		logger.Info(fmt.Sprintf("node.DesiredNodeState.Address: %s, host: %s, hostname: %s", node.DesiredNodeState.Address, host, hostname))
	}

	if node.cluster.Cmd.Kind == ckReplace {
		for host, hostname := range aliveHosts {
			logger.Info(fmt.Sprintf("newnode: %s host: %s hostname: %s", node.DesiredNodeState.Address, host, hostname))
			if err = newScriptExecuter(node.cluster.sshKey, node.DesiredNodeState.Address).
				run("node-update-hosts.sh", node.DesiredNodeState.Address, host, hostname); err != nil {
				logger.Error(err.Error())
				node.Error = err.Error()
				break
			}
		}
	}

	return err
}

func seNodeValidate(n *nodeType) error {
	prepareScripts("host-validate.sh")

	logger.Info(fmt.Sprintf("checking host %s requirements...", n.DesiredNodeState.Address))

	var minRAM string

	if skipNodeMemoryCheck {
		minRAM = "0"
	} else {
		minRAM = n.minAmountOfRAM()
	}

	var err error

	if err = newScriptExecuter(n.cluster.sshKey, n.DesiredNodeState.Address).
		run("host-validate.sh", n.DesiredNodeState.Address, minRAM); err != nil {
		logger.Error(err.Error())
		n.Error = err.Error()
	} else {
		logger.Info(fmt.Sprintf("host %s requirements checked successfully", n.DesiredNodeState.Address))
	}

	return err

}

func seClusterControllerFunction(c *clusterType) error {

	var err error

	switch c.Cmd.Kind {
	case ckInit:
		err = initSeCluster(c)
	case ckReplace:
		var n *nodeType
		if n = c.nodeByHost(c.Cmd.args()[1]); n == nil {
			return fmt.Errorf(ErrHostNotFoundInCluster.Error(), c.Cmd.args()[1])
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
	}
	return err
}

func initSeCluster(cluster *clusterType) error {
	var err error

	if err = deploySeSwarm(cluster); err != nil {
		logger.Error(err.Error)
		return err
	}

	if e := deployDbmsDockerStack(cluster); e != nil {
		logger.Error(e.Error)
		err = errors.Join(err, e)
	}

	if e := deploySeDockerStack(cluster); e != nil {
		logger.Error(e.Error)
		err = errors.Join(err, e)
	}

	if e := deployMonDockerStack(cluster); e != nil {
		logger.Error(e.Error)
		err = errors.Join(err, e)
	}

	return err
}

func deploySeSwarm(cluster *clusterType) error {

	var err error

	// Init swarm mode
	node := cluster.Nodes[idxSENode1]
	manager := node.nodeName() //ActualNodeState.Address

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

		logger.Info("swarm set label on", manager, node.label(swarmDbmsLabelKey))
		if err = newScriptExecuter(cluster.sshKey, manager).
			run("swarm-set-label.sh", manager, manager, node.label(swarmDbmsLabelKey), "true"); err != nil {
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

		if cluster.Nodes[i].nodeName() /*ActualNodeState.Address*/ == manager {
			continue
		}

		func(n *nodeType) {
			logger.Info("swarm add node on ", n.ActualNodeState.Address)
			if e := newScriptExecuter(cluster.sshKey, node.ActualNodeState.Address).
				run("swarm-add-node.sh", manager, n.nodeName() /*ActualNodeState.Address*/); e != nil {
				logger.Error(e.Error())
				n.Error = e.Error()
				return
			}

			logger.Info("swarm set label on", n.ActualNodeState.Address, n.label(swarmDbmsLabelKey))
			if e := newScriptExecuter(cluster.sshKey, n.ActualNodeState.Address).
				run("swarm-set-label.sh", manager, n.nodeName() /*ActualNodeState.Address*/, n.label(swarmDbmsLabelKey), "true"); e != nil {
				logger.Error(e.Error())
				n.Error = e.Error()
				return
			}

			logger.Info("db node prepare ", n.ActualNodeState.Address)
			if e := newScriptExecuter(cluster.sshKey, n.ActualNodeState.Address).
				run("db-node-prepare.sh", n.nodeName() /*ActualNodeState.Address*/); e != nil {
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

	if err := newScriptExecuter(cluster.sshKey, conf.AppNode1).
		// run("swarm-set-label.sh", conf.AppNode1, conf.AppNode1, cluster.nodeByHost(conf.AppNode1).label(swarmAppLabelKey), "true"); err != nil {
		run("swarm-set-label.sh", conf.AppNode1Name, conf.AppNode1Name, cluster.nodeByHost(conf.AppNode1).label(swarmAppLabelKey), "true"); err != nil {

		return err
	}

	if err := newScriptExecuter(cluster.sshKey, conf.AppNode2).
		//		run("swarm-set-label.sh", conf.AppNode1, conf.AppNode2, cluster.nodeByHost(conf.AppNode2).label(swarmAppLabelKey), "true"); err != nil {
		run("swarm-set-label.sh", conf.AppNode1Name, conf.AppNode2Name, cluster.nodeByHost(conf.AppNode2).label(swarmAppLabelKey), "true"); err != nil {

		return err
	}

	if err := newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s", conf.AppNode1, conf.AppNode2)).
		// run("se-cluster-start.sh", conf.AppNode1, conf.AppNode2); err != nil {
		run("se-cluster-start.sh", conf.AppNode1Name, conf.AppNode2Name); err != nil {

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
		//		run("db-cluster-start.sh", conf.DBNode1, conf.DBNode2, conf.DBNode3); err != nil {
		run("db-cluster-start.sh", conf.DBNode1Name, conf.DBNode2Name, conf.DBNode3Name); err != nil {
		return err
	}

	logger.Info("DBMS docker stack deployed successfully")
	return nil
}

func deployMonDockerStack(cluster *clusterType) error {

	logger.Info("Starting a mon docker stack deployment.")

	prepareScripts("alertmanager/config.yml", "prometheus/prometheus.yml", "prometheus/alert.rules",
		"docker-compose-mon.yml", "mon-node-prepare.sh", "mon-stack-start.sh", "swarm-set-label.sh")

	prepareScripts("grafana/grafana.ini",
		"grafana/provisioning/dashboards/swarmprom_dashboards.yml",
		"grafana/provisioning/dashboards/swarmprom-nodes-dash.json",
		"grafana/provisioning/dashboards/swarmprom-prometheus-dash.json",
		"grafana/provisioning/dashboards/swarmprom-services-dash.json",
		"grafana/provisioning/datasources/datasource.yml")

	conf := newSeConfigType(cluster)

	if err := newScriptExecuter(cluster.sshKey, conf.AppNode1).
		//run("swarm-set-label.sh", conf.AppNode1, conf.AppNode1, cluster.nodeByHost(conf.AppNode1).label(swarmMonLabelKey), "true"); err != nil {
		run("swarm-set-label.sh", conf.AppNode1Name, conf.AppNode1Name, cluster.nodeByHost(conf.AppNode1).label(swarmMonLabelKey), "true"); err != nil {
		return err
	}

	if err := newScriptExecuter(cluster.sshKey, conf.AppNode2).
		//run("swarm-set-label.sh", conf.AppNode1, conf.AppNode2, cluster.nodeByHost(conf.AppNode2).label(swarmMonLabelKey), "true"); err != nil {
		run("swarm-set-label.sh", conf.AppNode1Name, conf.AppNode2Name, cluster.nodeByHost(conf.AppNode2).label(swarmMonLabelKey), "true"); err != nil {
		return err
	}

	if err := newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s", conf.AppNode1, conf.AppNode2)).
		//run("mon-node-prepare.sh", conf.AppNode1, conf.AppNode2, conf.DBNode1, conf.DBNode2, conf.DBNode3); err != nil {
		run("mon-node-prepare.sh", conf.AppNode1Name, conf.AppNode2Name, conf.DBNode1Name, conf.DBNode2Name, conf.DBNode3Name); err != nil {

		return err
	}

	if err := newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s %s", conf.AppNode1, conf.AppNode2)).
		//run("mon-stack-start.sh", conf.AppNode1, conf.AppNode2); err != nil {
		run("mon-stack-start.sh", conf.AppNode1Name, conf.AppNode2Name); err != nil {
		return err
	}

	logger.Info("SE docker mon docker stack deployed successfully")
	return nil
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
}

func newSeConfigType(cluster *clusterType) *seConfigType {

	config := seConfigType{
		StackName: "voedger",
	}
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
	}
	return &config
}

func deployDocker(node *nodeType) error {
	var err error

	prepareScripts("docker-install.sh")

	logger.Info(fmt.Sprintf("deploy docker on a %s host...", node.DesiredNodeState.Address))

	if err = newScriptExecuter(node.cluster.sshKey, node.DesiredNodeState.Address).
		//		run("docker-install.sh", node.DesiredNodeState.Address); err != nil {
		run("docker-install.sh", node.nodeName()); err != nil {
		logger.Error(err.Error())
		node.Error = err.Error()
	} else {
		logger.Info("docker deployed successfully")
	}

	return err
}

func replaceSeScyllaNode(cluster *clusterType) error {
	var err error

	prepareScripts("ctool-scylla-replace-node.sh", "docker-install.sh", "swarm-add-node.sh",
		"db-node-prepare.sh", "db-bootstrap-prepare.sh", "swarm-rm-node.sh",
		"db-stack-update.sh", "docker-compose-template.yml", "swarm-set-label.sh", "docker-compose-prepare.sh",
		"scylla.yaml", "swarm-get-manager-token.sh")

	conf := newSeConfigType(cluster)

	if err = newScriptExecuter(cluster.sshKey, "localhost").
		run("swarm-get-manager-token.sh", conf.AppNode1); err != nil {
		logger.Error(err.Error())
		return err
	}

	oldAddr := cluster.Cmd.args()[0]
	newAddr := cluster.Cmd.args()[1]
	if conf.DBNode1 == newAddr {
		conf.DBNode1 = oldAddr
	} else if conf.DBNode2 == newAddr {
		conf.DBNode2 = oldAddr
	} else if conf.DBNode3 == newAddr {
		conf.DBNode3 = oldAddr
	}

	//	fmt.Println("docker-compose-prepare.sh", conf.DBNode1, conf.DBNode2, conf.DBNode3)
	fmt.Println("docker-compose-prepare.sh", conf.DBNode1Name, conf.DBNode2Name, conf.DBNode3Name)
	if err = newScriptExecuter(cluster.sshKey, "localhost").
		run("docker-compose-prepare.sh", conf.DBNode1Name, conf.DBNode2Name, conf.DBNode3Name); err != nil {
		logger.Error(err.Error())
		return err
	}

	if err = newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s, %s", oldAddr, newAddr)).
		run("ctool-scylla-replace-node.sh", oldAddr, newAddr, conf.AppNode1); err != nil {
		logger.Error(err.Error())
		return err
	}

	if err := newScriptExecuter(cluster.sshKey, newAddr).
		run("swarm-set-label.sh", conf.AppNode1, newAddr, cluster.nodeByHost(newAddr).label(swarmMonLabelKey), "true"); err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("sclylla node [%s -> %s] replaced successfully", oldAddr, newAddr))
	return nil
}

func replaceSeAppNode(cluster *clusterType) error {

	var err error

	prepareScripts("swarm-add-node.sh", "swarm-get-manager-token.sh", "swarm-rm-node.sh", "swarm-set-label.sh",
		"mon-node-prepare.sh", "prometheus-tsdb-copy.sh")

	prepareScripts("alertmanager/config.yml", "prometheus/prometheus.yml", "prometheus/alert.rules",
		"docker-compose-mon.yml")

	prepareScripts("grafana/grafana.ini",
		"grafana/provisioning/dashboards/swarmprom_dashboards.yml",
		"grafana/provisioning/dashboards/swarmprom-nodes-dash.json",
		"grafana/provisioning/dashboards/swarmprom-prometheus-dash.json",
		"grafana/provisioning/dashboards/swarmprom-services-dash.json",
		"grafana/provisioning/datasources/datasource.yml")

	conf := newSeConfigType(cluster)

	oldAddr := cluster.Cmd.args()[0]
	newAddr := cluster.Cmd.args()[1]

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

	if err = newScriptExecuter(cluster.sshKey, "localhost").
		run("swarm-get-manager-token.sh", conf.DBNode1); err != nil {
		logger.Error(err.Error())
		return err
	}

	logger.Info("swarm add node on ", newAddr)
	if err = newScriptExecuter(cluster.sshKey, newAddr).
		run("swarm-add-node.sh", conf.DBNode1, newAddr); err != nil {
		logger.Error(err.Error())
		return err
	}

	logger.Info("copy prometheus data base from", liveOldAddr, "to", newAddr)
	if err = newScriptExecuter(cluster.sshKey, fmt.Sprintf("%s, %s", liveOldAddr, newAddr)).
		run("prometheus-tsdb-copy.sh", liveOldAddr, newAddr); err != nil {
		logger.Error(err.Error())
		return err
	}

	logger.Info("mon node prepare ", newAddr)
	if err = newScriptExecuter(cluster.sshKey, newAddr).
		run("mon-node-prepare.sh", conf.AppNode1, conf.AppNode2, conf.DBNode1, conf.DBNode2, conf.DBNode3); err != nil {
		logger.Error(err.Error())
		return err
	}

	logger.Info("swarm set label on", newAddr, newNode.label(swarmAppLabelKey))
	if err = newScriptExecuter(cluster.sshKey, newAddr).
		run("swarm-set-label.sh", conf.DBNode1, newAddr, newNode.label(swarmDbmsLabelKey), "true"); err != nil {
		logger.Error(err.Error())
		return err
	}

	logger.Info("swarm set label on", newAddr, newNode.label(swarmMonLabelKey))
	if err = newScriptExecuter(cluster.sshKey, newAddr).
		run("swarm-set-label.sh", conf.DBNode1, newAddr, newNode.label(swarmMonLabelKey), "true"); err != nil {
		logger.Error(err.Error())
		return err
	}

	logger.Info("swarm remove node ", oldAddr)
	if err = newScriptExecuter(cluster.sshKey, oldAddr).
		run("swarm-rm-node.sh", conf.DBNode1, oldAddr); err != nil {
		logger.Error(err.Error())
		return err
	}

	return err
}
