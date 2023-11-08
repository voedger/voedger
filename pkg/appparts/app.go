/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
)

type app struct {
	name    istructs.AppQName
	storage istorage.IAppStorage
	// no locks need. Owned appPartitions structure will locks access to this structure
	parts map[istructs.PartitionID]*partition
}

func newApplication(name istructs.AppQName, storage istorage.IAppStorage) *app {
	return &app{
		name:    name,
		storage: storage,
		parts:   map[istructs.PartitionID]*partition{},
	}
}

type partition struct {
	app        *app
	appDef     appdef.IAppDef
	appStructs istructs.IAppStructs
	id         istructs.PartitionID
	pools      any
}

func newPartition(app *app, appDef appdef.IAppDef, appStructs istructs.IAppStructs, id istructs.PartitionID, pools any) *partition {
	p := &partition{app: app, appDef: appDef, appStructs: appStructs, id: id, pools: pools}
	return p
}

func (p *partition) borrow() (*partitionRT, error) {
	b := newPartitionRT(p)

	if err := b.init(); err != nil {
		return nil, err
	}

	return b, nil
}

type partitionRT struct {
	p          *partition
	appDef     appdef.IAppDef
	appStructs istructs.IAppStructs
}

func newPartitionRT(p *partition) *partitionRT {
	rt := &partitionRT{p: p, appDef: p.appDef, appStructs: p.appStructs}
	return rt
}

func (rt *partitionRT) App() istructs.AppQName           { return rt.p.app.name }
func (rt *partitionRT) AppStructs() istructs.IAppStructs { return rt.appStructs }
func (rt *partitionRT) ID() istructs.PartitionID         { return rt.p.id }
func (rt *partitionRT) Release()                         {}

// Initialize partition RT structures for use
func (rt *partitionRT) init() error { return nil }
