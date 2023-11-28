/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// ProcKind is a enumeration of processors.
//
// Ref to proc-king.go for values and methods
type ProcKind uint8

type IEngine interface{}

// Application partitions manager.
type IAppPartitions interface {
	// Adds new partition or update existing one.
	//
	// If partition with the same app and id already exists, it will be updated.
	//
	// @ConcurrentAccess
	Deploy(appName istructs.AppQName, partID istructs.PartitionID, appDef appdef.IAppDef, engines [ProcKind_Count][]IEngine)

	// Borrows and returns a partition.
	//
	// If partition not exist, returns error.
	//
	// @ConcurrentAccess
	Borrow(istructs.AppQName, istructs.PartitionID, ProcKind) (IAppPartition, IEngine, error)
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
