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

func (aps *appPartitions) AddOrReplace(appName istructs.AppQName, partID istructs.PartitionID, appDef appdef.IAppDef) {
	a := aps.apps[appName]
	if a == nil {
		s, err := aps.storages.AppStorage(appName)
		if err != nil {
			panic(err)
		}
		a = newApplication(appName, s)
		aps.apps[appName] = a
	}

	p := a.parts[partID]
	if p == nil {
		p = newPartition(a, appDef, partID)
		a.parts[partID] = p
	}
}

func (aps *appPartitions) Borrow(appName istructs.AppQName, partID istructs.PartitionID) (IAppPartition, error) {
	a := aps.apps[appName]
	if a == nil {
		return nil, fmt.Errorf(errAppNotFound, appName, ErrNameNotFound)
	}

	p := a.parts[partID]
	if p == nil {
		return nil, fmt.Errorf(errPartitionNotFound, appName, partID, ErrNameNotFound)
	}

	b, err := p.borrow()
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (aps *appPartitions) Release(p IAppPartition) {
	if rt, ok := p.(*partitionRT); ok {
		rt.p.release(rt)
	}
}
