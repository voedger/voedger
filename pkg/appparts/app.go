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
	parts   map[istructs.PartitionID]*partition
}

func newApplication(name istructs.AppQName, storage istorage.IAppStorage) *app {
	return &app{
		name:    name,
		storage: storage,
		parts:   map[istructs.PartitionID]*partition{},
	}
}

type partition struct {
	app      *app
	appDef   appdef.IAppDef
	id       istructs.PartitionID
	borrowed map[*partitionRT]bool
}

func newPartition(app *app, appDef appdef.IAppDef, id istructs.PartitionID) *partition {
	p := &partition{app: app, appDef: appDef, id: id, borrowed: map[*partitionRT]bool{}}
	return p
}

func (p *partition) borrow() (*partitionRT, error) {
	b := newPartitionRT(p)

	if err := b.prepare(); err != nil {
		return nil, err
	}

	p.borrowed[b] = true
	return b, nil
}

func (p *partition) release(borrowed *partitionRT) {
	delete(p.borrowed, borrowed)
}

type (
	partitionRT struct {
		p      *partition
		appDef appdef.IAppDef
	}
)

func newPartitionRT(p *partition) *partitionRT {
	rt := &partitionRT{p: p, appDef: p.appDef}
	return rt
}

func (rt *partitionRT) App() istructs.AppQName        { return rt.p.app.name }
func (rt *partitionRT) AppDef() appdef.IAppDef        { return rt.appDef }
func (rt *partitionRT) ID() istructs.PartitionID      { return rt.p.id }
func (rt *partitionRT) Storage() istorage.IAppStorage { return rt.p.app.storage }

// Prepares partition RT structures for use
func (rt *partitionRT) prepare() error { return nil }
