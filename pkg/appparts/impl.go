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

type appPartitions struct {
	storages istorage.IAppStorageProvider
	apps     map[istructs.AppQName]*app
}

func newAppPartitions(storages istorage.IAppStorageProvider) (ap IAppPartitions, cleanup func(), err error) {
	a := &appPartitions{
		storages: storages,
		apps:     map[istructs.AppQName]*app{},
	}
	return a, cleanup, err
}

func (aps appPartitions) AddApp(name istructs.AppQName, appDef appdef.IAppDef) error {
	s, err := aps.storages.AppStorage(name)
	if err != nil {
		return err
	}
	aps.apps[name] = newApplication(name, appDef, s)
	return nil
}

func (aps appPartitions) AddPartition(name istructs.AppQName, id istructs.PartitionID) error {
	a := aps.apps[name]
	if a == nil {
		return fmt.Errorf("app %s not found: %w", name, ErrNameNotFound)
	}
	a.parts[id] = newPartition(a, id)
	return nil
}

func (aps appPartitions) Borrow(name istructs.AppQName, id istructs.PartitionID) (IAppPartition, error) {
	return nil, nil
}

type appPartition struct {
	name    istructs.AppQName
	id      istructs.PartitionID
	appDef  appdef.IAppDef
	storage istorage.IAppStorage
}

func newAppPartition(name istructs.AppQName, id istructs.PartitionID, ad appdef.IAppDef, s istorage.IAppStorage) *appPartition {
	return &appPartition{name: name, id: id, appDef: ad, storage: s}
}

func (ap appPartition) AppName() istructs.AppQName { return ap.name }

func (ap appPartition) Partition() istructs.PartitionID { return ap.id }

func (ap appPartition) AppDef() appdef.IAppDef { return ap.appDef }

func (ap appPartition) Storage() istorage.IAppStorage { return ap.storage }

func (ap appPartition) prepare() error {
	return nil
}
