/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

// nolint
const (
	// edition types
	clusterEditionCE = "CE"
	clusterEditionSE = "SE"

	// permissions
	rwxrwxrwx = 0777
	rw_rw_rw_ = 0666

	// name of the cluster configuration file
	clusterConfFileName  = "cluster.json"
	scyllaConfigFileName = "scylla.yaml"

	shellLib = "utils.sh"

	// node fail version
	nodeFailVersion = "fail"

	// Deploy SE args
	deploySeFirstNodeArgIdx = 1
	initSeArgCount          = 5
	seNodeCount             = 2
	dbNodeCount             = 3

	// Deploy SE args
	initCeArgCount = 1

	// DB node offset in cluster node list
	dbNodeOffset = 2

	// node Roles
	nrCENode  = "CENode"
	nrAppNode = "AppNode"
	nrDBNode  = "DBNode"

	embedScriptsDir = "scripts"

	sshSockFile     = "/home/ubuntu/ssh-agent-sock"
	sshAgentPidFile = "/home/ubuntu/ssh-agent-pid"
)

const (
	// aliases for indexes of SE cluster nodes
	idxSENode1 = int32(iota)
	idxSENode2
	idxDBNode1
	idxDBNode2
	idxDBNode3
)

const (
	// command kind
	ckInit    = "init"
	ckUpgrade = "upgrade"
	ckReplace = "replace"

	// minimum amount of RAM per node in MB
	minRamOnAppNode = "8192"
	minRamOnDBNode  = "8192"
	minRamCENode    = "8192"

	swarmDbmsLabelKey = "dbm"
	swarmAppLabelKey  = "app"
	swarmMonLabelKey  = "mon"
)
