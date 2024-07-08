/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"
	"fmt"
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
func (a *app) deploy(def appdef.IAppDef /*extModuleURLs map[string]*url.URL*/, extensionModules []iextengine.ExtensionModule, structs istructs.IAppStructs, numEnginesPerEngineKind [ProcessorKind_Count]int) {
	a.def = def
	a.structs = structs

	eef := a.apps.extEngineFactories

	ctx := context.Background()

	// состоавить словарь extensionKind->path->iextengine.ExtensionModule
	// тут сначала сборать по всем путям все имена
	// extensionsPathsModules := map[appdef.ExtensionEngineKind]map[string]iextengine.ExtensionModule{}

	// def.PackageFullPath()
	// def.Extensions(func(i appdef.IExtension) {
	// 	extEngineKind := i.Engine()
	// 	// if i.Kind == Builtin then ModuleUrl i snot used
	// 	// else ModuleUrl is tkaen from extModuleURLs (panic here if msiing)

	// 	// посмотреть по path, нету ли iextengine.ExtensionModule, если нет, то создать, а если есть, то заапеенидить iextengine.ExtensionModule.ExtensionNames
	// 	extensionPathsModules, ok := extensionsPathsModules[extEngineKind]
	// 	if !ok {
	// 		extensionPathsModules = map[string]iextengine.ExtensionModule
	// 		extensionsPathsModules[extEngineKind] = extensionPathsModules
	// 	}
	// 	path := def.PackageFullPath(i.QName().Pkg())
	// 	extensionModule, ok := extensionPathsModules[path]
	// 	if !ok {
	// 		if extEngineKind != appdef.ExtensionEngineKind_BuiltIn {
	// 			moduleUrl, ok := extModuleURLs[path]
	// 			if !ok {
	// 				panic(fmt.Sprintf("app %s extension %s package url is not provided for path %s", a.name, i.QName(), path))
	// 			}
	// 			extensionModule.ModuleUrl = moduleUrl
	// 		}
	// 		extensionModule.Path = path
	// 	}
	// 	// FIXME: придумать как из IExtension взять имена
	// 	// extensionModule.ExtensionNames = append(extensionModule.ExtensionNames, nil)
	// 	extensionPathsModules[path] = extensionModule
	// })

	for k, cnt := range numEnginesPerEngineKind {
		extEngines := make([][]iextengine.IExtensionEngine, appdef.ExtensionEngineKind_Count)

		// TODO: prepare []iextengine.ExtensionPackage from IAppDef
		// TODO: should pass iextengine.ExtEngineConfig from somewhere (Provide?)
		// here run through IAppDef and: map[engineKind met among packages of IAppDef][]iextengine.ExtensionPackage
		// here extModuleURLs will be used on creating iextengine.ExtensionPackage
		// non-builtin -> has to be in extModuleURLs, panic otherwise

		// тут надо создавать только те движки, которые есть среди packages of IAppDef
		//
		def.Extensions(func(i appdef.IExtension) {
			extEngineKind := i.Engine()
			factory, ok := eef[extEngineKind] // factory тут - это либо для builtin, либо для wasm
			if !ok {
				panic(fmt.Errorf("no extension egine factory for engine %s met among def of %s", extEngineKind.String(), a.name))
			}
			factory.New(ctx, a.name, extensionModules, &iextengine.DefaultExtEngineConfig, 1) // FIXME: what is numEngines here?
		})

		// for ek, ef := range eef {

		// 	// non-native ->
		// 	ee, err := ef.New(ctx, a.name, []iextengine.ExtensionModule{
		// 		// тут заполнить
		// 	}, &iextengine.DefaultExtEngineConfig, cnt)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	extEngines[ek] = ee
		// }

		ee := make([]*engines, cnt)
		for i := 0; i < cnt; i++ {
			ee[i] = &engines{}
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
