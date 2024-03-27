/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts/internal/pool"
	"github.com/voedger/voedger/pkg/cluster"
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
	apps       *apps
	name       istructs.AppQName
	partsCount int
	def        appdef.IAppDef
	structs    istructs.IAppStructs
	engines    [cluster.ProcessorKind_Count]*pool.Pool[*engines]
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

	eef := a.apps.extEngineFactoriesFactory(a.name)

	ctx := context.Background()
	for k, cnt := range numEngines {
		extEngines := make([][]iextengine.IExtensionEngine, appdef.ExtensionEngineKind_Count)

		for ek, ef := range eef {
			ee, err := ef.New(ctx, []iextengine.ExtensionPackage{}, &iextengine.DefaultExtEngineConfig, cnt)
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

func (rt *partitionRT) App() istructs.AppQName { return rt.part.app.name }

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
		return errUndefinedExtension(name)
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
