/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// Application partitions manager.
type IAppPartitions interface {
	// Adds new partition or update existing one.
	//
	// If partition with the same app and id already exists, it will be updated.
	// @ConcurrentAccess
	AddOrReplace(appName istructs.AppQName, partID istructs.PartitionID, appDef appdef.IAppDef, pools any)

	// Borrows and returns a partition.
	//
	// If partition not exist, returns error.
	// @ConcurrentAccess
	Borrow(istructs.AppQName, istructs.PartitionID) (IAppPartition, error)
}

// Application partition.
type IAppPartition interface {
	App() istructs.AppQName
	ID() istructs.PartitionID

	AppStructs() istructs.IAppStructs

	// Releases borrowed partition.
	Release()
}
