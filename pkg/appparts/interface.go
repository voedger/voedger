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
	AddOrReplace(istructs.AppQName, istructs.PartitionID, appdef.IAppDef)

	Borrow(istructs.AppQName, istructs.PartitionID) (IAppPartition, error)
	Release(IAppPartition)
}

type IAppPartition interface {
	App() istructs.AppQName
	ID() istructs.PartitionID
	Storage() istorage.IAppStorage

	IAppPartRuntime
}

type IAppPartRuntime interface {
	AppDef() appdef.IAppDef
}
