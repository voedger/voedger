/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/istructs"
)

// Returns a new instance of IAppPartitionsController.
func New(parts appparts.IAppPartitions, apps []BuiltInApp) (ctl IAppPartitionsController, cleanup func(), err error) {
	return newAppPartitionsController(parts, apps)
}

// Describes built-in application.
type BuiltInApp struct {
	Name istructs.AppQName

	// Application definition will use to generate AppStructs
	Def appdef.IAppDef

	// Deployment parameters:

	// Number of partitions. Partitions IDs will be generated from 0 to NumParts-1
	NumParts int

	// EnginePoolSize pools size for each processor kind
	EnginePoolSize [cluster.ProcKind_Count]int
}
