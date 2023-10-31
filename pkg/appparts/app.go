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

type app struct {
	name    istructs.AppQName
	appDef  appdef.IAppDef
	storage istorage.IAppStorage
	parts   map[istructs.PartitionID]*partition
}

func newApplication(name istructs.AppQName, appDef appdef.IAppDef, storage istorage.IAppStorage) *app {
	return &app{
		name:    name,
		appDef:  appDef,
		storage: storage,
		parts:   map[istructs.PartitionID]*partition{},
	}
}

type partition struct {
	app *app
	id  istructs.PartitionID
}

func newPartition(app *app, id istructs.PartitionID) *partition {
	return &partition{app: app, id: id}
}
