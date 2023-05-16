/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import "errors"

var (
	ErrorDifferentNodeVersions              = errors.New("node versions do not match the cluster version")
	ErrorInvalidClusterEdition              = errors.New("invalid cluster edition (expected SE or CE)")
	ErrorInvalidNumberOfDataCenters         = errors.New("invalid number of data centers")
	ErrorClusterConfNotFound                = errors.New("cluster configuration not found\nuse the init command")
	ErrorClusterConfAlreadyExists           = errors.New("the cluster configuration already exists")
	ErrorInvalidNumberOfArguments           = errors.New("invalid number of arguments")
	ErrorInvalidIpAddress                   = errors.New("invalid IP-address")
	ErrorInvalidNodeRole                    = errors.New("invalid node role")
	ErrorUnknownCommand                     = errors.New("unknown command")
	ErrorMissingCommandArguments            = errors.New("missing command arguments")
	ErrorNoUpdgradeRequired                 = errors.New("no upgrade required")
	ErrorHostNotFoundInCluster              = errors.New("host %s not found in cluster")
	ErrorHostAlreadyExistsInCluster         = errors.New("host %s already exists in cluster")
	ErrorUncompletedCommandFound            = errors.New("uncompleted command found")
	ErrorNodeControllerFunctionNotAssigned  = errors.New("node controller function not assigned")
	ErrClusterControllerFunctionNotAssigned = errors.New("cluster controller function not assigned")
	ErrorPreparingClusterNodes              = errors.New("error preparing cluster nodes, the apply command is aborted")
)
