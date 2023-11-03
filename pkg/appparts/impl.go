/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"fmt"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
)

type appPartitions struct {
	storages istorage.IAppStorageProvider
	apps     map[istructs.AppQName]*app
	mx       sync.RWMutex
}

func newAppPartitions(storages istorage.IAppStorageProvider) (ap IAppPartitions, cleanup func(), err error) {
	a := &appPartitions{
		storages: storages,
		apps:     map[istructs.AppQName]*app{},
		mx:       sync.RWMutex{},
	}
	return a, func() {}, err
}

func (aps *appPartitions) AddOrUpdate(appName istructs.AppQName, partID istructs.PartitionID, appDef appdef.IAppDef) {
	aps.mx.Lock()
	defer aps.mx.Unlock()

	a, ok := aps.apps[appName]
	if !ok {
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

	p.appDef = appDef
}

func (aps *appPartitions) Borrow(appName istructs.AppQName, partID istructs.PartitionID) (IAppPartition, error) {
	aps.mx.RLock()
	defer aps.mx.RUnlock()

	a, ok := aps.apps[appName]
	if !ok {
		return nil, fmt.Errorf(errAppNotFound, appName, ErrNotFound)
	}

	p, ok := a.parts[partID]
	if !ok {
		return nil, fmt.Errorf(errPartitionNotFound, appName, partID, ErrNotFound)
	}

	b, err := p.borrow()
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (aps *appPartitions) Release(p IAppPartition) {
	if rt, ok := p.(*partitionRT); ok {
		aps.mx.Lock()
		defer aps.mx.Unlock()

		rt.p.release(rt)
	}
}
