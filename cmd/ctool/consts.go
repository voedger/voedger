/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

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

	// node fail version
	nodeFailVersion = "fail"

	// Deploy SE args
	deploySeFirstNodeArgIdx = 1
	initSeArgCount          = 5
	initSeWithDCArgCount    = 8
	seNodeCount             = 2
	dbNodeCount             = 3
	seDcCount               = 3

	// Deploy SE args
	initCeArgCount = 1

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
)

const (
	ceSettingsFileName = "ce-settings.json"
	seSettingsFileName = "se-settings.json"
)

const (
	swarmDbmsLabelKey = "dbm"
	swarmAppLabelKey  = "app"
	swarmMonLabelKey  = "mon"
)
