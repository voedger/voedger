/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"
	"fmt"
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
	byKind map[appdef.ExtensionEngineKind]iextengine.IExtensionEngine

	// there are 3 pools in partitionsRT: for query, command and actualizers.
	// this field need to return the current `engines` instance to the right one of these 3 pools
	pool *pool.Pool[*engines]
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

	// TODO: prepare []iextengine.ExtensionPackage from IAppDef
	// TODO: should pass iextengine.ExtEngineConfig from somewhere (Provide?)

	extModules := map[appdef.ExtensionEngineKind][]iextengine.ExtensionModule{}
	def.Extensions(func(ext appdef.IExtension) {
		extEngineKind := ext.Engine()
		path := ext.App().PackageFullPath(ext.QName().Pkg())
		moduleURL := extModuleURLs[path]
		extendionModule := iextengine.ExtensionModule{
			Path:           path,
			ModuleUrl:      moduleURL,
			ExtensionNames: []string{}, // TODO
		}
		extModules[extEngineKind] = append(extModules[extEngineKind], extendionModule)
	})

	// processorKind here is one of ProcessorKind_Command, ProcessorKind_Query, ProcessorKind_Actualizer
	for processorKind, processorsCountPerKind := range numEnginesPerEngineKind {
		ee := make([]*engines, processorsCountPerKind)
		for i := 0; i < processorsCountPerKind; i++ {
			for extEngineKind, extensionModules := range extModules {
				extensionEngineFactory, ok := eef[extEngineKind]
				if !ok {
					panic(fmt.Errorf("no extension engine factory for engine %s met among def of %s", extEngineKind.String(), a.name))
				}
				extEngines, err := extensionEngineFactory.New(ctx, a.name, extensionModules, &iextengine.DefaultExtEngineConfig, processorsCountPerKind) // FIXME: what is numEngines here?
				if err != nil {
					panic(err)
				}
				for i := 0; i < processorsCountPerKind; i++ {
					ee[i] = &engines{}
					ee[i].byKind[extEngineKind] = extEngines[i]
				}
			}
		}
		a.engines[processorKind] = pool.New(ee)
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
	// TODO: переделать тут: возвращать в Reelase откуда взяли, а не через engine.release, убрать engine.pool.
	pool := rt.part.app.engines[proc]
	engine, err := pool.Borrow() // will be released in (*engine).release()
	if err != nil {
		return errNotAvailableEngines[proc]
	}
	engine.pool = pool
	rt.borrowed = engine
	return nil
}
