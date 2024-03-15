/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"
	"errors"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts/internal/pool"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

// engine placeholder
type engine struct {
	iextengine.IExtensionEngine
	cluster.ProcessorKind
	pool *pool.Pool[*engine]
}

func newEngine(e iextengine.IExtensionEngine, kind cluster.ProcessorKind) *engine {
	return &engine{
		IExtensionEngine: e,
		ProcessorKind:    kind,
	}
}

func (e *engine) release() {
	if p := e.pool; p != nil {
		e.pool = nil
		p.Release(e)
	}
}

type app struct {
	apps       *apps
	name       istructs.AppQName
	partsCount int
	def        appdef.IAppDef
	structs    istructs.IAppStructs
	engines    [cluster.ProcessorKind_Count]*pool.Pool[*engine]
	// no locks need. Owned apps structure will locks access to this structure
	parts map[istructs.PartitionID]*partition
}

func newApplication(apps *apps, name istructs.AppQName, partsCount int) *app {
	return &app{
		apps:       apps,
		name:       name,
		partsCount: partsCount,
		parts:      map[istructs.PartitionID]*partition{},
	}
}

func (a *app) deploy(def appdef.IAppDef, structs istructs.IAppStructs, numEngines [cluster.ProcessorKind_Count]int) {
	a.def = def
	a.structs = structs

	ctx := context.Background()
	for k, cnt := range numEngines {
		// TODO: add support for WASM engine
		extEngines, err := a.apps.extEngineFactories[appdef.ExtensionEngineKind_BuiltIn].New(ctx, []iextengine.ExtensionPackage{}, nil, cnt)
		if err != nil {
			panic(err)
		}
		ee := make([]*engine, cnt)
		for i := 0; i < cnt; i++ {
			ee[i] = newEngine(extEngines[i], cluster.ProcessorKind(k))
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

func (p *partition) borrow(proc cluster.ProcessorKind) (*partitionRT, error) {
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
	borrowed   *engine
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

func (rt *partitionRT) App() istructs.AppQName           { return rt.part.app.name }
func (rt *partitionRT) AppStructs() istructs.IAppStructs { return rt.appStructs }
func (rt *partitionRT) ID() istructs.PartitionID         { return rt.part.id }

func (rt *partitionRT) Invoke(ctx context.Context, name appdef.QName, state istructs.IState, intents istructs.IIntents) error {
	// io := iextengine.NewExtensionIO(state, intents, rt.appDef, rt.appStructs)
	// extName := rt.app.appDef.FullQName(name)
	// if extName = appdef.NullFullQName {
	//    return undefinedExtension(name)
	// }
	// return rt.borrowed.Invoke(ctx, extName, io)
	return errors.ErrUnsupported
}

func (rt *partitionRT) Release() {
	if e := rt.borrowed; e != nil {
		rt.borrowed = nil
		e.release()
	}
	partionRTPool.Put(rt)
}

func (rt *partitionRT) DoSyncActualizer(ctx context.Context, work interface{}) error {
	return rt.part.syncActualizer.DoSync(ctx, work)
}

// Initialize partition RT structures for use
func (rt *partitionRT) init(proc cluster.ProcessorKind) error {
	pool := rt.part.app.engines[proc]
	engine, err := pool.Borrow() // will be released in (*engine).release()
	if err != nil {
		return errNotAvailableEngines[proc]
	}
	engine.pool = pool
	rt.borrowed = engine
	return nil
}
