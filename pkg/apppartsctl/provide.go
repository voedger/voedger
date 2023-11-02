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

// This can be passed to App() to enumerate range of partitions [min, max].
func PartsRange(min, max istructs.PartitionID) func(func(istructs.PartitionID)) {
	return func(f func(istructs.PartitionID)) {
		for i := min; i <= max; i++ {
			f(i)
		}
	}
}

// This can be passed to App() to enumerate partition IDs.
func PartsEnum(parts ...istructs.PartitionID) func(func(istructs.PartitionID)) {
	return func(f func(istructs.PartitionID)) {
		for _, p := range parts {
			f(p)
		}
	}
}

// Returns a new instance of IApplication to pass to New() for built-in applications.
//
// The parts argument is a function that enumerates all partitions of the application.
func App(name istructs.AppQName, appDef appdef.IAppDef, parts func(func(istructs.PartitionID))) IApplication {
	return newApplication(name, appDef, parts)
}

// Returns a new instance of IAppPartitionsController.
//
// Built-in applications can be constructed by calling App() and passed as optional arguments.
func New(storages istorage.IAppStorageProvider, builtIn ...IApplication) (ctl IAppPartitionsController, cleanup func(), err error) {
	return newAppPartitionsController(storages, builtIn...)
}
