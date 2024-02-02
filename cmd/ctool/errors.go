/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import "errors"

// nolint
var (
	ErrInvalidClusterEdition                = errors.New("invalid cluster edition (expected SE or CE)")
	ErrInvalidNumberOfDataCenters           = errors.New("invalid number of data centers")
	ErrClusterConfNotFound                  = errors.New("cluster configuration not found")
	ErrClusterConfAlreadyExists             = errors.New("the cluster configuration already exists")
	ErrInvalidNumberOfArguments             = errors.New("invalid number of arguments")
	ErrInvalidIpAddress                     = errors.New("invalid IP-address")
	ErrUnknownCommand                       = errors.New("unknown command")
	ErrMissingCommandArguments              = errors.New("missing command arguments")
	ErrNoUpdgradeRequired                   = errors.New("no upgrade required")
	ErrHostAlreadyExistsInCluster           = errors.New("host %s already exists in cluster")
	ErrUncompletedCommandFound              = errors.New("uncompleted command found")
	ErrNodeControllerFunctionNotAssigned    = errors.New("node controller function not assigned")
	ErrClusterControllerFunctionNotAssigned = errors.New("cluster controller function not assigned")
	ErrPreparingClusterNodes                = errors.New("error preparing cluster nodes, command is aborted")
	ErrManagerTokenNotExists                = errors.New("manager token not exists")
	ErrNoIncompleteCommandWasFoundToRepeat  = errors.New("no incomplete command was found to repeat")
)

var ErrIncorrectVersion = errors.New("incorrect version")

const (
	errCtoolVersionNewerThanClusterVersion = "ctool version %s is newer than cluster version %s: %w"
	errClusterVersionNewerThanCtoolVersion = "cluster version %s is newer than ctool version %s\nyou should use the ctool version %s: %w"
	errDifferentNodeVersion                = "node version %s do not match the cluster version %s: %w"
)

var ErrInvalidNodeRole = errors.New("invalid node role")

const errInvalidNodeRole = "node %s: %w"

var ErrEmptyNodeAddress = errors.New("empty IP-address")

const errEmptyNodeAddress = "node %s: %w"

// The commander cannot be executed
var ErrCommandCannotBeExecuted = errors.New("command cannot be executed")

const errCannotReplaceALiveNode = "cannot replace a live node %s: %w"

// host is not available
var ErrHostIsNotAvailable = errors.New("host is not available")

const errHostIsNotAvailable = "host %s is not available: %w"

var ErrAddressCannotBeUsed = errors.New("address cannot be used")

const errAddressInReplacedList = "address %s is in replaced list: %w"

var ErrHostNotFoundInCluster = errors.New("host not found")

const errHostNotFoundInCluster = "host %s not found in cluster: %w"

var ErrFileNotFound = errors.New("file not found")

const errSshKeyNotFound = "ssh key %s not found: %w"

var ErrDomainsNotFound = errors.New("domains not found")

const errDomainsNotFound = "domains %s not found in cluster: %w"
