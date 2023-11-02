/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
)

type app struct {
	name    istructs.AppQName
	appDef  appdef.IAppDef
	storage istorage.IAppStorage
	parts   map[istructs.PartitionID]*partition
}

func newApplication(name istructs.AppQName, appDef appdef.IAppDef, storage istorage.IAppStorage) *app {
	return &app{
		name:    name,
		appDef:  appDef,
		storage: storage,
		parts:   map[istructs.PartitionID]*partition{},
	}
}

type partition struct {
	app    *app
	id     istructs.PartitionID
	active bool
}

func newPartition(app *app, id istructs.PartitionID) *partition {
	p := &partition{app: app, id: id}
	app.parts[id] = p
	return p
}

func (p *partition) Activate() error {
	if p.active {
		return fmt.Errorf(errPartitionAlreadyActive, p.id, ErrInvalidPartitionStatus)
	}
	p.active = true
	return nil
}

func (p *partition) AppDef() appdef.IAppDef { return p.app.appDef }

func (p *partition) AppName() istructs.AppQName { return p.app.name }

func (p *partition) ID() istructs.PartitionID { return p.id }

func (p *partition) Active() bool { return p.active }

func (p *partition) Borrow() error {
	if !p.active {
		return fmt.Errorf(errPartitionIsInactive, p.id, ErrInvalidPartitionStatus)
	}
	return nil
}

func (p *partition) Storage() istorage.IAppStorage { return p.app.storage }
