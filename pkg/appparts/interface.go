/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/istructs"
)

// Application partitions manager.
type IAppPartitions interface {
	// Adds new application or update existing.
	//
	// If application with the same name exists, then its definition will be updated.
	//
	// @ConcurrentAccess
	DeployApp(appName istructs.AppQName, appDef appdef.IAppDef, engines [cluster.ProcessorKind_Count]int)

	// Deploys new partitions for specified application or update existing.
	//
	// If partition with the same app and id already exists, it will be updated.
	//
	// # Panics:
	// 	- if application not exists
	//
	// @ConcurrentAccess
	DeployAppPartitions(appName istructs.AppQName, partIDs []istructs.PartitionID)

	// Borrows and returns a partition.
	//
	// If partition not exist, returns error.
	//
	// @ConcurrentAccess
	Borrow(istructs.AppQName, istructs.PartitionID, cluster.ProcessorKind) (IAppPartition, error)
}

// Application partition.
type IAppPartition interface {
	App() istructs.AppQName
	ID() istructs.PartitionID

	AppStructs() istructs.IAppStructs

	// Releases borrowed partition.
	//
	// @ConcurrentAccess
	Release()
}
