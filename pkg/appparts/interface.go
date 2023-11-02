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

type IAppPartitions interface {
	AddApp(istructs.AppQName, appdef.IAppDef) error
	AddPartition(istructs.AppQName, istructs.PartitionID) error

	// Enums all partitions.
	Partitions(func(IAppPartition))
}

type IAppPartition interface {
	AppName() istructs.AppQName
	AppDef() appdef.IAppDef
	Storage() istorage.IAppStorage
	ID() istructs.PartitionID
	Active() bool

	// Activates partition. Initial status is Inactive, result status is Active.
	Activate() error

	// Borrows partition.
	Borrow() error
}
