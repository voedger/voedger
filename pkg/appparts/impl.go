/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type apps struct {
	structs               istructs.IAppStructsProvider
	syncActualizerFactory SyncActualizerFactory
	actualizers           IActualizers
	extEngineFactories    iextengine.ExtensionEngineFactories
	apps                  map[appdef.AppQName]*app
	mx                    sync.RWMutex
}

func newAppPartitions(asp istructs.IAppStructsProvider, saf SyncActualizerFactory, act IActualizers, eef iextengine.ExtensionEngineFactories) (ap IAppPartitions, cleanup func(), err error) {
	a := &apps{
		structs:               asp,
		syncActualizerFactory: saf,
		actualizers:           act,
		extEngineFactories:    eef,
		apps:                  map[appdef.AppQName]*app{},
		mx:                    sync.RWMutex{},
	}
	return a, func() {}, err
}

func (aps *apps) DeployApp(name appdef.AppQName, def appdef.IAppDef, partsCount istructs.NumAppPartitions, engines [ProcessorKind_Count]int) {
	aps.mx.Lock()
	defer aps.mx.Unlock()

	if _, ok := aps.apps[name]; ok {
		panic(errAppCannotBeRedeployed(name))
	}

	a := newApplication(aps, name, partsCount)
	aps.apps[name] = a

	appStructs, err := aps.structs.BuiltIn(name)
	if err != nil {
		panic(err)
	}

	a.deploy(def, appStructs, engines)
}

func (aps *apps) DeployAppPartitions(appName appdef.AppQName, partIDs []istructs.PartitionID) {
	aps.mx.Lock()
	defer aps.mx.Unlock()

	a, ok := aps.apps[appName]
	if !ok {
		panic(errAppNotFound(appName))
	}

	for _, id := range partIDs {
		p := newPartition(a, id)
		a.parts[id] = p
	}

	var err error
	for _, id := range partIDs {
		if e := aps.actualizers.DeployPartition(appName, id); e != nil {
			err = errors.Join(err, e)
		}
	}

	if err != nil {
		panic(err)
	}
}

func (aps *apps) AppDef(appName appdef.AppQName) (appdef.IAppDef, error) {
	aps.mx.Lock()
	defer aps.mx.Unlock()

	app, ok := aps.apps[appName]
	if !ok {
		return nil, errAppNotFound(appName)
	}
	return app.def, nil
}

// Returns _total_ application partitions count.
//
// This is a configuration value for the application, independent of how many sections are currently deployed.
func (aps *apps) AppPartsCount(appName appdef.AppQName) (istructs.NumAppPartitions, error) {
	aps.mx.Lock()
	defer aps.mx.Unlock()

	app, ok := aps.apps[appName]
	if !ok {
		return 0, errAppNotFound(appName)
	}
	return app.partsCount, nil
}

func (aps *apps) Borrow(appName appdef.AppQName, partID istructs.PartitionID, proc ProcessorKind) (IAppPartition, error) {
	aps.mx.RLock()
	defer aps.mx.RUnlock()

	app, ok := aps.apps[appName]
	if !ok {
		return nil, errAppNotFound(appName)
	}

	part, ok := app.parts[partID]
	if !ok {
		return nil, errPartitionNotFound(appName, partID)
	}

	borrowed, err := part.borrow(proc)
	if err != nil {
		return nil, err
	}

	return borrowed, nil
}

func (aps *apps) AppWorkspacePartitionID(appName appdef.AppQName, ws istructs.WSID) (istructs.PartitionID, error) {
	pc, err := aps.AppPartsCount(appName)
	if err != nil {
		return 0, err
	}
	return coreutils.AppPartitionID(ws, pc), nil
}

func (aps *apps) WaitForBorrow(ctx context.Context, appName appdef.AppQName, partID istructs.PartitionID, proc ProcessorKind) (IAppPartition, error) {
	for ctx.Err() == nil {
		ap, err := aps.Borrow(appName, partID, proc)
		if err == nil {
			return ap, nil
		}
		if errors.Is(err, ErrNotAvailableEngines) {
			time.Sleep(AppPartitionBorrowRetryDelay)
			continue
		}
		return nil, err
	}
	return nil, ctx.Err()
}
