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
type engines map[appdef.ExtensionEngineKind]iextengine.IExtensionEngine

type app struct {
	mx           sync.RWMutex
	apps         *apps
	name         appdef.AppQName
	partsCount   istructs.NumAppPartitions
	def          appdef.IAppDef
	structs      istructs.IAppStructs
	enginesPools [ProcessorKind_Count]*pool.Pool[engines]
	parts        map[istructs.PartitionID]*partition
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

	enginesPathsModules := map[appdef.ExtensionEngineKind]map[string]*iextengine.ExtensionModule{}
	def.Extensions(func(ext appdef.IExtension) {
		extEngineKind := ext.Engine()
		path := ext.App().PackageFullPath(ext.QName().Pkg())
		pathsModules, ok := enginesPathsModules[extEngineKind]
		if !ok {
			// initialize any engine mentioned in the schema
			pathsModules = map[string]*iextengine.ExtensionModule{}
			enginesPathsModules[extEngineKind] = pathsModules
		}
		if extEngineKind != appdef.ExtensionEngineKind_WASM {
			return
		}
		extModule, ok := pathsModules[path]
		if !ok {
			moduleURL, ok := extModuleURLs[path]
			if !ok {
				panic(fmt.Sprintf("module path %s is missing among extension modules URLs", path))
			}
			extModule = &iextengine.ExtensionModule{
				Path:      path,
				ModuleUrl: moduleURL,
			}
			pathsModules[path] = extModule
		}
		extModule.ExtensionNames = append(extModule.ExtensionNames, ext.QName().Entity())
	})
	extModules := map[appdef.ExtensionEngineKind][]iextengine.ExtensionModule{}
	for extEngineKind, pathsModules := range enginesPathsModules {
		extModules[extEngineKind] = nil // initialize any engine mentioned in the schema
		for _, extModule := range pathsModules {
			extModules[extEngineKind] = append(extModules[extEngineKind], *extModule)
		}
	}

	// processorKind here is one of ProcessorKind_Command, ProcessorKind_Query, ProcessorKind_Actualizer
	for processorKind, processorsCountPerKind := range numEnginesPerEngineKind {
		ee := make([]engines, processorsCountPerKind)
		for extEngineKind, extensionModules := range extModules {
			extensionEngineFactory, ok := eef[extEngineKind]
			if !ok {
				panic(fmt.Errorf("no extension engine factory for engine %s met among def of %s", extEngineKind.String(), a.name))
			}
			extEngines, err := extensionEngineFactory.New(ctx, a.name, extensionModules, &iextengine.DefaultExtEngineConfig, processorsCountPerKind)
			if err != nil {
				panic(err)
			}
			for i := 0; i < processorsCountPerKind; i++ {
				if ee[i] == nil {
					ee[i] = map[appdef.ExtensionEngineKind]iextengine.IExtensionEngine{}
				}
				ee[i][extEngineKind] = extEngines[i]
			}
		}
		a.enginesPools[processorKind] = pool.New(ee)
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
	part                     *partition
	appDef                   appdef.IAppDef
	appStructs               istructs.IAppStructs
	borrowed                 engines
	borrowedForProcessorKind ProcessorKind
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

func (rt *partitionRT) DoSyncActualizer(ctx context.Context, work pipeline.IWorkpiece) error {
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

	extEngine, ok := rt.borrowed[e.Engine()]
	if !ok {
		return fmt.Errorf("no extension engine for extension kind %s", e.Engine().String())
	}

	return extEngine.Invoke(ctx, extName, io)
}

func (rt *partitionRT) Release() {
	if engine := rt.borrowed; engine != nil {
		rt.borrowed = nil
		poolTheEngineBorrowedFrom := rt.part.app.enginesPools[rt.borrowedForProcessorKind]
		poolTheEngineBorrowedFrom.Release(engine)
		rt.borrowedForProcessorKind = ProcessorKind_Count // like null
	}
	partionRTPool.Put(rt)
}

// Initialize partition RT structures for use
func (rt *partitionRT) init(proc ProcessorKind) error {
	pool := rt.part.app.enginesPools[proc]
	engines, err := pool.Borrow()
	if err != nil {
		return errNotAvailableEngines[proc]
	}
	rt.borrowed = engines
	rt.borrowedForProcessorKind = proc
	return nil
}
