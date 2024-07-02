/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"
	"net/url"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts/internal/pool"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

// engine placeholder
type engines struct {
	byKind [appdef.ExtensionEngineKind_Count]iextengine.IExtensionEngine
	pool   *pool.Pool[*engines]
}

func newEngines() *engines {
	return &engines{}
}

func (e *engines) release() {
	if p := e.pool; p != nil {
		e.pool = nil
		p.Release(e)
	}
}

type app struct {
	mx         sync.RWMutex
	apps       *apps
	name       appdef.AppQName
	partsCount istructs.NumAppPartitions
	def        appdef.IAppDef
	structs    istructs.IAppStructs
	engines    [ProcessorKind_Count]*pool.Pool[*engines]
	parts      map[istructs.PartitionID]*partition
}

func newApplication(apps *apps, name appdef.AppQName, partsCount istructs.NumAppPartitions) *app {
	return &app{
		mx:         sync.RWMutex{},
		apps:       apps,
		name:       name,
		partsCount: partsCount,
		parts:      map[istructs.PartitionID]*partition{},
	}
}

// extModuleURLs is important for non-builtin (non-native) apps
// extModuleURLs: packagePath->packageURL
func (a *app) deploy(def appdef.IAppDef, extModuleURLs map[string]*url.URL, structs istructs.IAppStructs, numEnginesPerEngineKind [ProcessorKind_Count]int) {
	a.def = def
	a.structs = structs

	eef := a.apps.extEngineFactories

	ctx := context.Background()
	// тут надо создавать только те движки, которые есть среди packages of IAppDef
	for k, cnt := range numEnginesPerEngineKind {
		extEngines := make([][]iextengine.IExtensionEngine, appdef.ExtensionEngineKind_Count)

		// TODO: prepare []iextengine.ExtensionPackage from IAppDef
		// TODO: should pass iextengine.ExtEngineConfig from somewhere (Provide?)
		// here run through IAppDef and: map[engineKind met among packages of IAppDef][]iextengine.ExtensionPackage
		// here extModuleURLs will be used on creating iextengine.ExtensionPackage
		// non-builtin -> has to be in extModuleURLs, panic otherwise
		for ek, ef := range eef {

			// non-native ->
			ee, err := ef.New(ctx, a.name, []iextengine.ExtensionPackage{
				// тут заполнить
			}, &iextengine.DefaultExtEngineConfig, cnt)
			if err != nil {
				panic(err)
			}
			extEngines[ek] = ee
		}

		ee := make([]*engines, cnt)
		for i := 0; i < cnt; i++ {
			ee[i] = newEngines()
			for ek := range eef {
				ee[i].byKind[ek] = extEngines[ek][i]
			}
		}
		a.engines[k] = pool.New(ee)
	}
}

type partition struct {
	app            *app
	id             istructs.PartitionID
	syncActualizer pipeline.ISyncOperator
}

func newPartition(app *app, id istructs.PartitionID) *partition {
	part := &partition{
		app:            app,
		id:             id,
		syncActualizer: app.apps.syncActualizerFactory(app.structs, id),
	}
	return part
}

func (p *partition) borrow(proc ProcessorKind) (*partitionRT, error) {
	b := newPartitionRT(p)

	if err := b.init(proc); err != nil {
		return nil, err
	}

	return b, nil
}

type partitionRT struct {
	part       *partition
	appDef     appdef.IAppDef
	appStructs istructs.IAppStructs
	borrowed   *engines
}

var partionRTPool = sync.Pool{
	New: func() interface{} {
		return &partitionRT{}
	},
}

func newPartitionRT(part *partition) *partitionRT {
	rt := partionRTPool.Get().(*partitionRT)

	rt.part = part
	rt.appDef = part.app.def
	rt.appStructs = part.app.structs

	return rt
}

func (rt *partitionRT) App() appdef.AppQName { return rt.part.app.name }

func (rt *partitionRT) AppStructs() istructs.IAppStructs { return rt.appStructs }

func (rt *partitionRT) DoSyncActualizer(ctx context.Context, work interface{}) error {
	return rt.part.syncActualizer.DoSync(ctx, work)
}

func (rt *partitionRT) ID() istructs.PartitionID { return rt.part.id }

func (rt *partitionRT) Invoke(ctx context.Context, name appdef.QName, state istructs.IState, intents istructs.IIntents) error {
	e := rt.appDef.Extension(name)
	if e == nil {
		return errUndefinedExtension(name)
	}

	extName := rt.appDef.FullQName(name)
	if extName == appdef.NullFullQName {
		return errCantObtainFullQName(name)
	}
	io := iextengine.NewExtensionIO(rt.appDef, state, intents)

	return rt.borrowed.byKind[e.Engine()].Invoke(ctx, extName, io)
}

func (rt *partitionRT) Release() {
	if e := rt.borrowed; e != nil {
		rt.borrowed = nil
		e.release()
	}
	partionRTPool.Put(rt)
}

// Initialize partition RT structures for use
func (rt *partitionRT) init(proc ProcessorKind) error {
	pool := rt.part.app.engines[proc]
	engine, err := pool.Borrow() // will be released in (*engine).release()
	if err != nil {
		return errNotAvailableEngines[proc]
	}
	engine.pool = pool
	rt.borrowed = engine
	return nil
}
