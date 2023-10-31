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
	Borrow(istructs.AppQName, istructs.PartitionID) (IAppPartition, error)
}

type IAppPartition interface {
	AppName() istructs.AppQName
	Partition() istructs.PartitionID
	AppDef() appdef.IAppDef
	Storage() istorage.IAppStorage
}
