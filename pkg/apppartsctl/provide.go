/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
)

// Returns a new instance of IApplication to pass to New() for built-in applications.
//
// The parts argument is a function that enumerates all partitions of the application.
func NewApp(name istructs.AppQName, appDef appdef.IAppDef, parts func(func(istructs.PartitionID))) IApplication {
	return newApplication(name, appDef, parts)
}

// Returns a new instance of IAppPartitionsController.
//
// Built-in applications can be passed as optional arguments.
func New(storages istorage.IAppStorageProvider, builtIn ...IApplication) (ctl IAppPartitionsController, cleanup func(), err error) {
	return newAppPartitionsController(storages, builtIn...)
}
