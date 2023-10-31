/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iservices"
	"github.com/voedger/voedger/pkg/istructs"
)

type IAppPartitions interface {
	iservices.IService
}

type IAppPartition interface {
	AppDef() appdef.IAppDef
}

type IAppPartitionsAPI interface {
	Get(istructs.AppQName) IAppPartition
}
