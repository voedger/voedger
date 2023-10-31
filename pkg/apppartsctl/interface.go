/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iservices"
	"github.com/voedger/voedger/pkg/istructs"
)

// IApplication is an application that consists of partitions.
type IApplication interface {
	AppName() istructs.AppQName
	AppDef() appdef.IAppDef

	// Enums all partitions of the application.
	Partitions(func(istructs.PartitionID))
}

// IAppPartitionsController is a service that creates, updates (replaces) and deletes applications partitions.
type IAppPartitionsController interface {
	iservices.IService
}
