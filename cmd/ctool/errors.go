/*
* Copyright (c) 2023-present Sigma-Soft, Ltd.
* @author Dmitry Molchanovsky
 */

package main

import "errors"

var (
	ErrorDifferentNodeVersions      = errors.New("node versions do not match the cluster version")
	ErrorInvalidClusterEdition      = errors.New("invalid cluster edition (expected SE or CE)")
	ErrorInvalidNumberOfDataCenters = errors.New("invalid number of data centers")
	ErrorClusterConfNotFound        = errors.New("cluster configuration not found\nuse the init command")
	ErrorClusterConfAlreadyExists   = errors.New("the cluster configuration already exists")
	ErrorInvalidNumberOfArguments   = errors.New("invalid number of arguments")
	ErrorInvalidIpAddress           = errors.New("invalid IP-address")
	ErrorInvalidNodeRole            = errors.New("invalid node role")
)
