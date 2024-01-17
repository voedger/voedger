/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts/internal/pool"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/istructs"
)

// engine placeholder
type engine struct {
	cluster.ProcessorKind
	pool *pool.Pool[*engine]
}

func newEngine(kind cluster.ProcessorKind) *engine {
	return &engine{
		ProcessorKind: kind,
	}
}

func (e *engine) release() {
	if p := e.pool; p != nil {
		e.pool = nil
		p.Release(e)
	}
}

type app struct {
	name       istructs.AppQName
	def        appdef.IAppDef
	partsCount int
	structs    istructs.IAppStructs
	engines    [cluster.ProcessorKind_Count]*pool.Pool[*engine]
	// no locks need. Owned apps structure will locks access to this structure
	parts map[istructs.PartitionID]*partition
}

func newApplication(name istructs.AppQName) *app {
	return &app{
		name:  name,
		parts: map[istructs.PartitionID]*partition{},
	}
}

func (a *app) deploy(def appdef.IAppDef, structs istructs.IAppStructs, partsCount int, engines [cluster.ProcessorKind_Count]int) {
	a.def = def
	a.structs = structs
	a.partsCount = partsCount
	for k, cnt := range engines {
		ee := make([]*engine, cnt)
		for i := 0; i < cnt; i++ {
			ee[i] = newEngine(cluster.ProcessorKind(k))
		}
		a.engines[k] = pool.New[*engine](ee)
	}
}

type partition struct {
	app *app
	id  istructs.PartitionID
}

func newPartition(app *app, id istructs.PartitionID) *partition {
	part := &partition{
		app: app,
		id:  id,
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

func newPartitionRT(part *partition) *partitionRT {
	rt := &partitionRT{
		part:       part,
		appDef:     part.app.def,
		appStructs: part.app.structs,
	}
	return rt
}

func (rt *partitionRT) App() istructs.AppQName           { return rt.part.app.name }
func (rt *partitionRT) AppStructs() istructs.IAppStructs { return rt.appStructs }
func (rt *partitionRT) ID() istructs.PartitionID         { return rt.part.id }

func (rt *partitionRT) Release() {
	if e := rt.borrowed; e != nil {
		rt.borrowed = nil
		e.release()
	}
}

// Initialize partition RT structures for use
func (rt *partitionRT) init(proc cluster.ProcessorKind) error {
	pool := rt.part.app.engines[proc]
	engine, err := pool.Borrow() // will be released in (*engine).release()
	if err != nil {
		return fmt.Errorf(errNotEnoughEngines, proc.TrimString(), err)
	}
	engine.pool = pool
	rt.borrowed = engine
	return nil
}
