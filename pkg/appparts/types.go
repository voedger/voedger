/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"net/url"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// ProcessorKind is a enumeration of processors.
// moved here from pkg/cluster to avoid import cycle: appparts uses cluster.ProcessorKid, cluster uses appparts.IAppPartitions in c.cluster.AppDeploy
type ProcessorKind uint8

//go:generate stringer -type=ProcessorKind

const (
	ProcessorKind_Command ProcessorKind = iota
	ProcessorKind_Query
	ProcessorKind_Actualizer
	ProcessorKind_Scheduler

	ProcessorKind_Count
)

type AppDeploymentDescriptor struct {
	// Number of partitions. Partitions IDs will be generated from 0 to NumParts-1
	//
	// NumParts should contain _total_ number of partitions, not only to deploy.
	NumParts istructs.NumAppPartitions

	// EnginePoolSize pools size for each processor kind
	EnginePoolSize [ProcessorKind_Count]uint

	// total number of AppWorkspaces
	NumAppWorkspaces istructs.NumAppWorkspaces
}

func PoolSize(c, q, p, s uint) [ProcessorKind_Count]uint { return [ProcessorKind_Count]uint{c, q, p, s} }

// Describes built-in application.
type BuiltInApp struct {
	AppDeploymentDescriptor

	Name appdef.AppQName

	// Application definition will use to generate AppStructs
	Def appdef.IAppDef
}

type SidecarApp struct {
	BuiltInApp
	ExtModuleURLs map[string]*url.URL
}
