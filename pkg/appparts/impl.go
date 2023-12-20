/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"fmt"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/istructs"
)

type apps struct {
	structs istructs.IAppStructsProvider
	apps    map[istructs.AppQName]*app
	mx      sync.RWMutex
}

func newAppPartitions(structs istructs.IAppStructsProvider) (ap IAppPartitions, cleanup func(), err error) {
	a := &apps{
		structs: structs,
		apps:    map[istructs.AppQName]*app{},
		mx:      sync.RWMutex{},
	}
	return a, func() {}, err
}

func (aps *apps) DeployApp(appName istructs.AppQName, appDef appdef.IAppDef, engines [cluster.ProcessorKind_Count]int) {
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

	a.deploy(appDef, appStructs, engines)
}

func (aps *apps) DeployAppPartitions(appName istructs.AppQName, partIDs []istructs.PartitionID) {
	aps.mx.Lock()
	defer aps.mx.Unlock()

	a, ok := aps.apps[appName]
	if !ok {
		panic(fmt.Errorf(errAppNotFound, appName, ErrNotFound))
	}

	for _, id := range partIDs {
		p := newPartition(a, id)
		a.parts[id] = p
	}
}

func (aps *apps) Borrow(appName istructs.AppQName, partID istructs.PartitionID, proc cluster.ProcessorKind) (IAppPartition, error) {
	aps.mx.RLock()
	defer aps.mx.RUnlock()

	app, ok := aps.apps[appName]
	if !ok {
		return nil, fmt.Errorf(errAppNotFound, appName, ErrNotFound)
	}

	part, ok := app.parts[partID]
	if !ok {
		return nil, fmt.Errorf(errPartitionNotFound, appName, partID, ErrNotFound)
	}

	borrowed, err := part.borrow(proc)
	if err != nil {
		return nil, err
	}

	return borrowed, nil
}
