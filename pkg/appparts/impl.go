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
		return fmt.Errorf(errAppNotFound, name, ErrNameNotFound)
	}
	a.parts[id] = newPartition(a, id)
	return nil
}

func (aps appPartitions) Borrow(name istructs.AppQName, id istructs.PartitionID) (IAppPartition, error) {
	a, ok := aps.apps[name]
	if !ok {
		return nil, fmt.Errorf(errAppNotFound, name, ErrNameNotFound)
	}
	p, ok := a.parts[id]
	if !ok {
		return nil, fmt.Errorf(errPartitionNotFound, id, ErrNameNotFound)
	}
	return p, nil
}
