/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts/internal/pool"
	"github.com/voedger/voedger/pkg/istructs"
)

type app struct {
	name    istructs.AppQName
	def     appdef.IAppDef
	structs istructs.IAppStructs
	engines [ProcKind_Count]*pool.Pool[IEngine]
	// no locks need. Owned apps structure will locks access to this structure
	parts map[istructs.PartitionID]*partition
}

func newApplication(name istructs.AppQName) *app {
	return &app{
		name:  name,
		parts: map[istructs.PartitionID]*partition{},
	}
}

func (a *app) deploy(def appdef.IAppDef, structs istructs.IAppStructs, engines [ProcKind_Count][]IEngine) error {
	a.def = def
	a.structs = structs
	for k, ee := range engines {
		a.engines[k] = pool.New[IEngine](ee)
	}
	return nil
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

func (p *partition) borrow(proc ProcKind) (*partitionRT, error) {
	b := newPartitionRT(p, proc)

	if err := b.init(proc); err != nil {
		return nil, err
	}

	return b, nil
}

type partitionRT struct {
	part       *partition
	appDef     appdef.IAppDef
	appStructs istructs.IAppStructs
	borrowed   struct {
		engine IEngine
		pool   *pool.Pool[IEngine]
	}
}

func newPartitionRT(part *partition, proc ProcKind) *partitionRT {
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
	if e := rt.borrowed.engine; e != nil {
		rt.borrowed.engine = nil
		rt.borrowed.pool.Release(e)
	}
}

// Initialize partition RT structures for use
func (rt *partitionRT) init(proc ProcKind) error {
	engine, err := rt.part.app.engines[proc].Borrow()
	if err != nil {
		return fmt.Errorf(errNotEnoughEngines, proc.TrimString(), err)
	}
	rt.borrowed.engine = engine
	rt.borrowed.pool = rt.part.app.engines[proc]
	return nil
}
