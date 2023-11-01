/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import "errors"

// nolint
var (
	ErrDifferentNodeVersions                = errors.New("node versions do not match the cluster version")
	ErrInvalidClusterEdition                = errors.New("invalid cluster edition (expected SE or CE)")
	ErrInvalidNumberOfDataCenters           = errors.New("invalid number of data centers")
	ErrClusterConfNotFound                  = errors.New("cluster configuration not found\nuse the init command")
	ErrClusterConfAlreadyExists             = errors.New("the cluster configuration already exists")
	ErrInvalidNumberOfArguments             = errors.New("invalid number of arguments")
	ErrInvalidIpAddress                     = errors.New("invalid IP-address")
	ErrInvalidNodeRole                      = errors.New("invalid node role")
	ErrUnknownCommand                       = errors.New("unknown command")
	ErrMissingCommandArguments              = errors.New("missing command arguments")
	ErrNoUpdgradeRequired                   = errors.New("no upgrade required")
	ErrHostNotFoundInCluster                = errors.New("host %s not found in cluster")
	ErrHostAlreadyExistsInCluster           = errors.New("host %s already exists in cluster")
	ErrUncompletedCommandFound              = errors.New("uncompleted command found")
	ErrNodeControllerFunctionNotAssigned    = errors.New("node controller function not assigned")
	ErrClusterControllerFunctionNotAssigned = errors.New("cluster controller function not assigned")
	ErrPreparingClusterNodes                = errors.New("error preparing cluster nodes, command is aborted")
	ErrManagerTokenNotExists                = errors.New("manager token not exists")
)
