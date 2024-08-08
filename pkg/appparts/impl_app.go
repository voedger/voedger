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

type engines map[appdef.ExtensionEngineKind]iextengine.IExtensionEngine

type appVersion struct {
	def     appdef.IAppDef
	structs istructs.IAppStructs
	pools   [ProcessorKind_Count]*pool.Pool[engines]
}

type appRT struct {
	mx             sync.RWMutex
	apps           *apps
	name           appdef.AppQName
	partsCount     istructs.NumAppPartitions
	lastestVersion appVersion
	parts          map[istructs.PartitionID]*appPartitionRT
}

func newApplication(apps *apps, name appdef.AppQName, partsCount istructs.NumAppPartitions) *appRT {
	return &appRT{
		mx:             sync.RWMutex{},
		apps:           apps,
		name:           name,
		partsCount:     partsCount,
		lastestVersion: appVersion{},
		parts:          map[istructs.PartitionID]*appPartitionRT{},
	}
}

// extModuleURLs is important for non-builtin (non-native) apps
// extModuleURLs: packagePath->packageURL
func (a *appRT) deploy(def appdef.IAppDef, extModuleURLs map[string]*url.URL, structs istructs.IAppStructs, numEnginesPerEngineKind [ProcessorKind_Count]int) {
	a.lastestVersion.def = def
	a.lastestVersion.structs = structs

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
		a.lastestVersion.pools[processorKind] = pool.New(ee)
	}
}

type appPartitionRT struct {
	app            *appRT
	id             istructs.PartitionID
	syncActualizer pipeline.ISyncOperator
	// TODO: implement partitionCache
}

func newAppPartitionRT(app *appRT, id istructs.PartitionID) *appPartitionRT {
	part := &appPartitionRT{
		app:            app,
		id:             id,
		syncActualizer: app.apps.syncActualizerFactory(app.lastestVersion.structs, id),
	}
	return part
}

func (p *appPartitionRT) borrow(proc ProcessorKind) (*borrowedPartition, error) {
	b := newBorrowedPartition(p)

	if err := b.init(proc); err != nil {
		return nil, err
	}

	return b, nil
}

// # Implements IAppPartition
type borrowedPartition struct {
	part       *appPartitionRT
	appDef     appdef.IAppDef
	appStructs istructs.IAppStructs
	engines    engines
	procKind   ProcessorKind
}

var borrowedPartitionsPool = sync.Pool{
	New: func() interface{} {
		return &borrowedPartition{}
	},
}

func newBorrowedPartition(part *appPartitionRT) *borrowedPartition {
	rt := borrowedPartitionsPool.Get().(*borrowedPartition)

	rt.part = part
	rt.appDef = part.app.lastestVersion.def
	rt.appStructs = part.app.lastestVersion.structs

	return rt
}

// # IAppPartition.App
func (rt *borrowedPartition) App() appdef.AppQName { return rt.part.app.name }

// # IAppPartition.AppStructs
func (rt *borrowedPartition) AppStructs() istructs.IAppStructs { return rt.appStructs }

// # IAppPartition.DoSyncActualizer
func (rt *borrowedPartition) DoSyncActualizer(ctx context.Context, work pipeline.IWorkpiece) error {
	return rt.part.syncActualizer.DoSync(ctx, work)
}

// # IAppPartition.ID
func (rt *borrowedPartition) ID() istructs.PartitionID { return rt.part.id }

// # IAppPartition.Invoke
func (rt *borrowedPartition) Invoke(ctx context.Context, name appdef.QName, state istructs.IState, intents istructs.IIntents) error {
	e := rt.appDef.Extension(name)
	if e == nil {
		return errUndefinedExtension(name)
	}

	extName := rt.appDef.FullQName(name)
	if extName == appdef.NullFullQName {
		return errCantObtainFullQName(name)
	}
	io := iextengine.NewExtensionIO(rt.appDef, state, intents)

	extEngine, ok := rt.engines[e.Engine()]
	if !ok {
		return fmt.Errorf("no extension engine for extension kind %s", e.Engine().String())
	}

	return extEngine.Invoke(ctx, extName, io)
}

// # IAppPartition.Release
func (rt *borrowedPartition) Release() {
	if engine := rt.engines; engine != nil {
		rt.engines = nil
		pool := rt.part.app.lastestVersion.pools[rt.procKind] // source pool the engine borrowed from
		pool.Release(engine)
	}
	borrowedPartitionsPool.Put(rt)
}

// Initialize borrowed partition structures for use
func (rt *borrowedPartition) init(proc ProcessorKind) error {
	pool := rt.part.app.lastestVersion.pools[proc]
	engines, err := pool.Borrow()
	if err != nil {
		return errNotAvailableEngines[proc]
	}
	rt.engines = engines
	rt.procKind = proc
	return nil
}
