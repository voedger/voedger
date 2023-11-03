/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
)

// Application partitions manager.
type IAppPartitions interface {
	// Adds new partition or update existing one.
	//
	// If partition with the same app and id already exists, it will be updated.
	AddOrUpdate(istructs.AppQName, istructs.PartitionID, appdef.IAppDef)

	// Borrows and returns a partition.
	//
	// If partition with the given app and id does not exist, returns error.
	Borrow(istructs.AppQName, istructs.PartitionID) (IAppPartition, error)

	// Releases borrowed partition.
	Release(IAppPartition)
}

// Application partition.
type IAppPartition interface {
	App() istructs.AppQName
	ID() istructs.PartitionID
	Storage() istorage.IAppStorage

	IAppPartRuntime
}

// Application partition runtime.
type IAppPartRuntime interface {
	AppDef() appdef.IAppDef
}
