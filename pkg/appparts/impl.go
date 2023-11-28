/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"fmt"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type appPartitions struct {
	structs istructs.IAppStructsProvider
	apps    map[istructs.AppQName]*app
	mx      sync.RWMutex
}

func newAppPartitions(structs istructs.IAppStructsProvider) (ap IAppPartitions, cleanup func(), err error) {
	a := &appPartitions{
		structs: structs,
		apps:    map[istructs.AppQName]*app{},
		mx:      sync.RWMutex{},
	}
	return a, func() {}, err
}

func (aps *appPartitions) Deploy(appName istructs.AppQName, partID []istructs.PartitionID, appDef appdef.IAppDef, engines [ProcKind_Count][]IEngine) {
	aps.mx.Lock()
	defer aps.mx.Unlock()

	a, ok := aps.apps[appName]
	if !ok {
		a = newApplication(appName)
		aps.apps[appName] = a
	}

	appStructs, err := aps.structs.AppStructsByDef(appName, appDef)
	if err != nil {
		panic(err)
	}
	for _, id := range partID {
		p := newPartition(a, appDef, appStructs, id, engines)
		a.parts[id] = p
	}
}

func (aps *appPartitions) Borrow(appName istructs.AppQName, partID istructs.PartitionID, proc ProcKind) (IAppPartition, IEngine, error) {
	aps.mx.RLock()
	defer aps.mx.RUnlock()

	app, ok := aps.apps[appName]
	if !ok {
		return nil, nil, fmt.Errorf(errAppNotFound, appName, ErrNotFound)
	}

	part, ok := app.parts[partID]
	if !ok {
		return nil, nil, fmt.Errorf(errPartitionNotFound, appName, partID, ErrNotFound)
	}

	borrowed, engine, err := part.borrow(proc)
	if err != nil {
		return nil, nil, err
	}

	return borrowed, engine, nil
}
