/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
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

	// Number of partitions. Partitions IDs will be generated from 1 to NumParts
	NumParts int

	// Engines for each processor kind
	Engines [appparts.ProcKind_Count][]appparts.IEngine
}
