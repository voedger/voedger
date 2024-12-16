/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"
	"errors"
	"iter"
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istructs"
)

type apps struct {
	mx                     sync.RWMutex
	vvmCtx                 context.Context
	structs                istructs.IAppStructsProvider
	syncActualizerFactory  SyncActualizerFactory
	asyncActualizersRunner IActualizerRunner
	schedulerRunner        ISchedulerRunner
	extEngineFactories     iextengine.ExtensionEngineFactories
	bucketsFactory         irates.BucketsFactoryType
	apps                   map[appdef.AppQName]*appRT
}

func newAppPartitions(
	vvmCtx context.Context,
	asp istructs.IAppStructsProvider,
	saf SyncActualizerFactory,
	asyncActualizersRunner IActualizerRunner,
	jobSchedulerRunner ISchedulerRunner,
	eef iextengine.ExtensionEngineFactories,
	bf irates.BucketsFactoryType,
) (ap IAppPartitions, cleanup func(), err error) {
	a := &apps{
		mx:                     sync.RWMutex{},
		vvmCtx:                 vvmCtx,
		structs:                asp,
		asyncActualizersRunner: asyncActualizersRunner,
		schedulerRunner:        jobSchedulerRunner,
		syncActualizerFactory:  saf,
		extEngineFactories:     eef,
		bucketsFactory:         bf,
		apps:                   map[appdef.AppQName]*appRT{},
	}
	a.asyncActualizersRunner.SetAppPartitions(a)
	a.schedulerRunner.SetAppPartitions(a)
	return a, func() {}, err
}

func (aps *apps) AppDef(name appdef.AppQName) (appdef.IAppDef, error) {
	aps.mx.RLock()
	app, ok := aps.apps[name]
	aps.mx.RUnlock()

	if !ok {
		return nil, errAppNotFound(name)
	}
	return app.lastestVersion.appDef(), nil
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

func (aps *apps) AppWorkspacePartitionID(name appdef.AppQName, ws istructs.WSID) (istructs.PartitionID, error) {
	pc, err := aps.AppPartsCount(name)
	if err != nil {
		return 0, err
	}
	return coreutils.AppPartitionID(ws, pc), nil
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

func (aps *apps) DeployApp(name appdef.AppQName, extModuleURLs map[string]*url.URL, def appdef.IAppDef,
	partsCount istructs.NumAppPartitions, engines [ProcessorKind_Count]uint, numAppWorkspaces istructs.NumAppWorkspaces) {
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

	var appStructs istructs.IAppStructs
	var err error
	if len(extModuleURLs) != 0 {
		// TODO: is sidecarapp criteria?
		appStructs, err = aps.structs.New(name, def, istructs.ClusterApps[name], numAppWorkspaces)
	} else {
		appStructs, err = aps.structs.BuiltIn(name)
	}
	if err != nil {
		panic(err)
	}

	a.deploy(def, extModuleURLs, appStructs, engines)
}

func (aps *apps) DeployAppPartitions(name appdef.AppQName, ids []istructs.PartitionID) {
	aps.mx.RLock()
	a, ok := aps.apps[name]
	aps.mx.RUnlock()

	if !ok {
		panic(errAppNotFound(name))
	}

	wg := sync.WaitGroup{}
	for _, id := range ids {
		var p *appPartitionRT

		a.mx.Lock()
		if exists, ok := a.parts[id]; ok {
			p = exists
		} else {
			p = newAppPartitionRT(a, id)
			a.parts[id] = p
		}
		a.mx.Unlock()

		wg.Add(1)
		go func(p *appPartitionRT) {
			p.actualizers.Deploy(
				aps.vvmCtx,
				a.lastestVersion.appDef(),
				aps.asyncActualizersRunner.NewAndRun,
			)
			wg.Done()
		}(p)

		wg.Add(1)
		go func(p *appPartitionRT) {
			p.schedulers.Deploy(
				aps.vvmCtx,
				a.lastestVersion.appDef(),
				aps.schedulerRunner.NewAndRun,
			)
			wg.Done()
		}(p)
	}
	wg.Wait()
}

func (aps *apps) UpgradeAppDef(name appdef.AppQName, def appdef.IAppDef) {
	aps.mx.RLock()
	app, ok := aps.apps[name]
	aps.mx.RUnlock()

	if !ok {
		panic(errAppNotFound(name))
	}

	app.lastestVersion.upgrade(def, app.lastestVersion.appStructs(), app.lastestVersion.pools)
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

func (aps *apps) WorkedActualizers(app appdef.AppQName) iter.Seq2[istructs.PartitionID, []appdef.QName] {
	parts := []istructs.PartitionID{}
	names := map[istructs.PartitionID]appdef.QNames{}

	aps.mx.RLock()
	if a, ok := aps.apps[app]; ok {
		for id, p := range a.parts {
			if actualizers := p.actualizers.Enum(); len(actualizers) != 0 {
				parts = append(parts, id)
				names[id] = actualizers
			}
		}
	}
	aps.mx.RUnlock()

	slices.Sort(parts)

	return func(visit func(istructs.PartitionID, []appdef.QName) bool) {
		for _, id := range parts {
			if !visit(id, names[id]) {
				return
			}
		}
	}
}

func (aps *apps) WorkedSchedulers(app appdef.AppQName) iter.Seq2[istructs.PartitionID, map[appdef.QName][]istructs.WSID] {
	parts := []istructs.PartitionID{}
	names := map[istructs.PartitionID]map[appdef.QName][]istructs.WSID{}

	aps.mx.RLock()
	if a, ok := aps.apps[app]; ok {
		for id, p := range a.parts {
			if schedulers := p.schedulers.Enum(); len(schedulers) != 0 {
				parts = append(parts, id)
				names[id] = p.schedulers.Enum()
			}
		}
	}
	aps.mx.RUnlock()

	slices.Sort(parts)

	return func(visit func(istructs.PartitionID, map[appdef.QName][]istructs.WSID) bool) {
		for _, id := range parts {
			if !visit(id, names[id]) {
				return
			}
		}
	}
}
