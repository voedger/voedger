/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/acl"
	"github.com/voedger/voedger/pkg/appparts/internal/actualizers"
	"github.com/voedger/voedger/pkg/appparts/internal/limiter"
	"github.com/voedger/voedger/pkg/appparts/internal/pool"
	"github.com/voedger/voedger/pkg/appparts/internal/schedulers"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

type engines map[appdef.ExtensionEngineKind]iextengine.IExtensionEngine

type appVersion struct {
	mx      sync.RWMutex
	def     appdef.IAppDef
	structs istructs.IAppStructs
	pools   [ProcessorKind_Count]*pool.Pool[engines]
}

func (av *appVersion) appDef() appdef.IAppDef {
	av.mx.RLock()
	defer av.mx.RUnlock()
	return av.def
}

func (av *appVersion) appStructs() istructs.IAppStructs {
	av.mx.RLock()
	defer av.mx.RUnlock()
	return av.structs
}

// returns AppDef, AppStructs and engines pool for the specified processor kind
func (av *appVersion) snapshot(proc ProcessorKind) (appdef.IAppDef, istructs.IAppStructs, *pool.Pool[engines]) {
	av.mx.RLock()
	defer av.mx.RUnlock()
	return av.def, av.structs, av.pools[proc]
}

func (av *appVersion) upgrade(
	def appdef.IAppDef,
	structs istructs.IAppStructs,
	pools [ProcessorKind_Count]*pool.Pool[engines],
) {
	av.mx.Lock()
	defer av.mx.Unlock()

	av.def = def
	av.structs = structs
	av.pools = pools
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
func (a *appRT) deploy(def appdef.IAppDef, extModuleURLs map[string]*url.URL, structs istructs.IAppStructs, numEnginesPerEngineKind [ProcessorKind_Count]uint) {
	eef := a.apps.extEngineFactories

	if err := a.validateExtensions(def, eef); err != nil {
		panic(err)
	}

	enginesPathsModules := map[appdef.ExtensionEngineKind]map[string]*iextengine.ExtensionModule{}
	for ext := range appdef.Extensions(def.Types()) {
		extEngineKind := ext.Engine()
		path := ext.App().PackageFullPath(ext.QName().Pkg())
		pathsModules, ok := enginesPathsModules[extEngineKind]
		if !ok {
			// initialize any engine mentioned in the schema
			pathsModules = map[string]*iextengine.ExtensionModule{}
			enginesPathsModules[extEngineKind] = pathsModules
		}
		if extEngineKind != appdef.ExtensionEngineKind_WASM {
			continue
		}
		extModule, ok := pathsModules[path]
		if !ok {
			moduleURL, ok := extModuleURLs[path]
			if !ok {
				panic(fmt.Sprintf("module path %s is missing among extension modules URLs", path))
			}
			extModule = &iextengine.ExtensionModule{
				Path:      path,
				ModuleURL: moduleURL,
			}
			pathsModules[path] = extModule
		}
		extModule.ExtensionNames = append(extModule.ExtensionNames, ext.QName().Entity())
	}
	extModules := map[appdef.ExtensionEngineKind][]iextengine.ExtensionModule{}
	for extEngineKind, pathsModules := range enginesPathsModules {
		extModules[extEngineKind] = nil // initialize any engine mentioned in the schema
		for _, extModule := range pathsModules {
			extModules[extEngineKind] = append(extModules[extEngineKind], *extModule)
		}
	}

	pools := [ProcessorKind_Count]*pool.Pool[engines]{}
	// processorKind here is one of ProcessorKind_Command, ProcessorKind_Query, ProcessorKind_Actualizer, ProcessorKind_Scheduler
	for processorKind, processorsCountPerKind := range numEnginesPerEngineKind {
		ee := make([]engines, processorsCountPerKind)
		for extEngineKind, extensionModules := range extModules {
			extensionEngineFactory, ok := eef[extEngineKind]
			if !ok {
				panic(fmt.Errorf("no extension engine factory for engine %s met among def of %s", extEngineKind.String(), a.name))
			}
			extEngines, err := extensionEngineFactory.New(a.apps.vvmCtx, a.name, extensionModules, &iextengine.DefaultExtEngineConfig, processorsCountPerKind)
			if err != nil {
				panic(errExtensionEngineDeploy(a.name, extEngineKind, err))
			}
			for i := uint(0); i < processorsCountPerKind; i++ {
				if ee[i] == nil {
					ee[i] = map[appdef.ExtensionEngineKind]iextengine.IExtensionEngine{}
				}
				ee[i][extEngineKind] = extEngines[i]
			}
		}
		pools[processorKind] = pool.New(ee)
	}

	a.lastestVersion.upgrade(def, structs, pools)
}

// builtInFuncsRegistry is implemented by the BuiltIn extension engine factory and
// gives the deployment-time validator access to the merged per-app and stateless
// BuiltInExtFuncs maps. Matched via duck typing to avoid widening the public
// iextengine surface.
type builtInFuncsRegistry interface {
	AppFuncs() iextengine.BuiltInAppExtFuncs
	StatelessFuncs() iextengine.BuiltInExtFuncs
}

// validateExtensions cross-checks vsql-declared extensions against code-registered
// implementations for the BuiltIn engine, in both directions:
//   - in vsql, not in code: every BuiltIn extension declared in def must have an
//     entry either in the per-app or stateless BuiltInExtFuncs map
//   - in code, not in vsql: every per-app entry, and every stateless entry whose
//     package path is known to def, must be visited during the AppDef walk
//
// WASM extensions are validated by wazero.initModule when the engine factory is
// constructed; this function leaves them to that path.
func (a *appRT) validateExtensions(def appdef.IAppDef, eef iextengine.ExtensionEngineFactories) error {
	registry, ok := eef[appdef.ExtensionEngineKind_BuiltIn].(builtInFuncsRegistry)
	if !ok {
		return nil
	}

	appFuncs := registry.AppFuncs()[a.name]
	statelessFuncs := registry.StatelessFuncs()

	visited := map[appdef.FullQName]bool{}
	var errs []error
	for ext := range appdef.Extensions(def.Types()) {
		if ext.Engine() != appdef.ExtensionEngineKind_BuiltIn {
			continue
		}
		fqn := def.FullQName(ext.QName())
		if fqn == appdef.NullFullQName {
			errs = append(errs, errExtensionUnknownPackage(a.name, ext))
			continue
		}
		if _, ok := appFuncs[fqn]; ok {
			visited[fqn] = true
			continue
		}
		if _, ok := statelessFuncs[fqn]; ok {
			visited[fqn] = true
			continue
		}
		errs = append(errs, errExtensionInVSQLNotInCode(a.name, ext, fqn))
	}

	for fqn := range appFuncs {
		if !visited[fqn] {
			errs = append(errs, errExtensionInCodeNotInVSQL(a.name, fqn))
		}
	}
	for fqn := range statelessFuncs {
		if visited[fqn] {
			continue
		}
		if def.PackageLocalName(fqn.PkgPath()) == "" {
			continue
		}
		errs = append(errs, errExtensionInCodeNotInVSQL(a.name, fqn))
	}

	return errors.Join(errs...)
}

type appPartitionRT struct {
	app            *appRT
	id             istructs.PartitionID
	syncActualizer pipeline.ISyncOperator
	actualizers    *actualizers.PartitionActualizers
	schedulers     *schedulers.PartitionSchedulers
	limiter        *limiter.Limiter
}

func newAppPartitionRT(app *appRT, id istructs.PartitionID) *appPartitionRT {
	as := app.lastestVersion.appStructs()
	buckets := app.apps.bucketsFactory()
	part := &appPartitionRT{
		app:            app,
		id:             id,
		syncActualizer: app.apps.syncActualizerFactory(as, id),
		actualizers:    actualizers.New(app.name, id),
		schedulers:     schedulers.New(app.name, app.partsCount, as.NumAppWorkspaces(), id),
		limiter:        limiter.New(app.lastestVersion.appDef(), buckets),
	}
	return part
}

func (p *appPartitionRT) borrow(proc ProcessorKind) (*borrowedPartition, error) {
	b := newBorrowedPartition(p)

	if err := b.borrow(proc); err != nil {
		b.Release()
		return nil, err
	}

	return b, nil
}

// # Supports:
//   - IAppPartition
type borrowedPartition struct {
	part       *appPartitionRT
	appDef     appdef.IAppDef
	appStructs istructs.IAppStructs
	pool       *pool.Pool[engines] // pool of borrowed engines
	kind       ProcessorKind
	engines    engines // borrowed engines
}

var borrowedPartitionsPool = sync.Pool{
	New: func() interface{} {
		return &borrowedPartition{}
	},
}

func newBorrowedPartition(part *appPartitionRT) *borrowedPartition {
	bp := borrowedPartitionsPool.Get().(*borrowedPartition)
	bp.part = part
	return bp
}

// # IAppPartition.App
func (bp *borrowedPartition) App() appdef.AppQName { return bp.part.app.name }

// # IAppPartition.AppStructs
func (bp *borrowedPartition) AppStructs() istructs.IAppStructs { return bp.appStructs }

// # IAppPartition.DoSyncActualizer
func (bp *borrowedPartition) DoSyncActualizer(ctx context.Context, work pipeline.IWorkpiece) error {
	return bp.part.syncActualizer.DoSync(ctx, work)
}

// # IAppPartition.ID
func (bp *borrowedPartition) ID() istructs.PartitionID { return bp.part.id }

// # IAppPartition.Invoke
func (bp *borrowedPartition) Invoke(ctx context.Context, name appdef.QName, state istructs.IState, intents istructs.IIntents) error {
	e := appdef.Extension(bp.appDef.Type, name)
	if e == nil {
		return errUndefinedExtension(name)
	}

	if compat, err := bp.kind.CompatibleWithExtension(e); !compat {
		return fmt.Errorf("%s: %w", bp, err)
	}

	extName := bp.appDef.FullQName(name)
	if extName == appdef.NullFullQName {
		return errCantObtainFullQName(name)
	}
	io := iextengine.NewExtensionIO(bp.appDef, state, intents)

	extEngine, ok := bp.engines[e.Engine()]
	if !ok {
		return fmt.Errorf("no extension engine for extension kind %s", e.Engine().String())
	}

	return extEngine.Invoke(ctx, extName, io)
}

func (bp *borrowedPartition) IsLimitExceeded(resource appdef.QName, operation appdef.OperationKind, workspace istructs.WSID, remoteAddr string) (bool, appdef.QName) {
	return bp.part.limiter.Exceeded(resource, operation, workspace, remoteAddr)
}

func (bp *borrowedPartition) ResetRateLimit(resource appdef.QName, operation appdef.OperationKind, workspace istructs.WSID, remoteAddr string) {
	bp.part.limiter.ResetLimits(resource, operation, workspace, remoteAddr)
}

func (bp *borrowedPartition) IsOperationAllowed(ws appdef.IWorkspace, op appdef.OperationKind, res appdef.QName, fld []appdef.FieldName, roles []appdef.QName) (bool, error) {
	return acl.IsOperationAllowed(ws, op, res, fld, roles)
}

func (bp *borrowedPartition) String() string {
	return fmt.Sprintf("borrowedPartition{app=%s, part=%d, kind=%s}", bp.part.app.name, bp.part.id, bp.kind)
}

// # IAppPartition.Release
func (bp *borrowedPartition) Release() {
	bp.part = nil
	bp.appDef = nil
	bp.appStructs = nil
	if pool := bp.pool; pool != nil {
		bp.pool = nil
		if engine := bp.engines; engine != nil {
			bp.engines = nil
			pool.Release(engine)
		}
	}
	borrowedPartitionsPool.Put(bp)
}

func (bp *borrowedPartition) borrow(proc ProcessorKind) (err error) {
	bp.kind = proc
	bp.appDef, bp.appStructs, bp.pool = bp.part.app.lastestVersion.snapshot(proc)
	bp.engines, err = bp.pool.Borrow()
	if err != nil {
		return errNotAvailableEngines[proc]
	}
	return nil
}
