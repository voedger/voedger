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
	mx                    sync.RWMutex
	structs               istructs.IAppStructsProvider
	syncActualizerFactory SyncActualizerFactory
	actualizers           IActualizers
	extEngineFactories    iextengine.ExtensionEngineFactories
	apps                  map[appdef.AppQName]*app
}

func newAppPartitions(asp istructs.IAppStructsProvider, saf SyncActualizerFactory, act IActualizers, eef iextengine.ExtensionEngineFactories) (ap IAppPartitions, cleanup func(), err error) {
	a := &apps{
		mx:                    sync.RWMutex{},
		structs:               asp,
		syncActualizerFactory: saf,
		actualizers:           act,
		extEngineFactories:    eef,
		apps:                  map[appdef.AppQName]*app{},
	}
	act.SetAppPartitions(a)
	return a, func() {}, err
}

func (aps *apps) DeployApp(name appdef.AppQName, def appdef.IAppDef, partsCount istructs.NumAppPartitions, engines [ProcessorKind_Count]int) {
	aps.mx.RLock()
	_, ok := aps.apps[name]
	aps.mx.RUnlock()

	if ok {
		panic(errAppCannotBeRedeployed(name))
	}

	a := newApplication(aps, name, partsCount)
	aps.mx.Lock()
	aps.apps[name] = a
	aps.mx.Unlock()

	appStructs, err := aps.structs.BuiltIn(name)
	if err != nil {
		panic(err)
	}

	a.deploy(def, appStructs, engines)
}

func (aps *apps) DeployAppPartitions(name appdef.AppQName, ids []istructs.PartitionID) {
	aps.mx.RLock()
	a, ok := aps.apps[name]
	aps.mx.RUnlock()

	if !ok {
		panic(errAppNotFound(name))
	}

	//TODO: parallelize
	for _, id := range ids {
		p := newPartition(a, id)
		a.mx.Lock()
		a.parts[id] = p
		a.mx.Unlock()
	}

	var err error
	for _, id := range ids {
		if e := aps.actualizers.DeployPartition(name, id); e != nil {
			err = errors.Join(err, e)
		}
	}

	if err != nil {
		panic(err)
	}
}

func (aps *apps) AppDef(name appdef.AppQName) (appdef.IAppDef, error) {
	aps.mx.RLock()
	app, ok := aps.apps[name]
	aps.mx.RUnlock()

	if !ok {
		return nil, errAppNotFound(name)
	}
	return app.def, nil
}

// Returns _total_ application partitions count.
//
// This is a configuration value for the application, independent of how many sections are currently deployed.
func (aps *apps) AppPartsCount(name appdef.AppQName) (istructs.NumAppPartitions, error) {
	aps.mx.RLock()
	app, ok := aps.apps[name]
	aps.mx.RUnlock()

	if !ok {
		return 0, errAppNotFound(name)
	}
	return app.partsCount, nil
}

func (aps *apps) Borrow(name appdef.AppQName, id istructs.PartitionID, proc ProcessorKind) (IAppPartition, error) {
	aps.mx.RLock()
	app, ok := aps.apps[name]
	aps.mx.RUnlock()

	if !ok {
		return nil, errAppNotFound(name)
	}

	app.mx.RLock()
	part, ok := app.parts[id]
	app.mx.RUnlock()

	if !ok {
		return nil, errPartitionNotFound(name, id)
	}

	borrowed, err := part.borrow(proc)
	if err != nil {
		return nil, err
	}

	return borrowed, nil
}

func (aps *apps) AppWorkspacePartitionID(name appdef.AppQName, ws istructs.WSID) (istructs.PartitionID, error) {
	pc, err := aps.AppPartsCount(name)
	if err != nil {
		return 0, err
	}
	return coreutils.AppPartitionID(ws, pc), nil
}

func (aps *apps) WaitForBorrow(ctx context.Context, name appdef.AppQName, id istructs.PartitionID, proc ProcessorKind) (IAppPartition, error) {
	for ctx.Err() == nil {
		ap, err := aps.Borrow(name, id, proc)
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
