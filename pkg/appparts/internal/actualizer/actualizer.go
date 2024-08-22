/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package actualizer

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type ActualizerRun func(context.Context, appdef.AppQName, istructs.PartitionID, appdef.QName)

type Actualizers struct {
	mx          sync.RWMutex
	app         appdef.AppQName
	part        istructs.PartitionID
	actualizers map[appdef.QName]*actualizerRT
}

func newActualizers(app appdef.AppQName, part istructs.PartitionID) *Actualizers {
	return &Actualizers{
		app:         app,
		part:        part,
		actualizers: make(map[appdef.QName]*actualizerRT),
	}
}

// Deploys partition actualizers: stops actualizers for removed projectors and
// starts actualizers for new projectors
func (pa *Actualizers) Deploy(vvmCtx context.Context, appDef appdef.IAppDef, run ActualizerRun) {
	// async stop old processors
	stopWG := sync.WaitGroup{}
	pa.mx.RLock()
	for name, rt := range pa.actualizers {
		// TODO: compare if projector properties changed (events, sync/async, etc.)
		if appDef.Projector(name) == nil {
			stopWG.Add(1)
			go func(rt *actualizerRT) {
				rt.cancel()
				for rt.state.Load() >= 0 {
					time.Sleep(time.Nanosecond) // wait until actualizer is finished
				}
				stopWG.Done()
			}(rt)
		}
	}
	pa.mx.RUnlock()
	stopWG.Wait() // wait for all old actualizers to stop

	// async start new actualizers
	startWG := sync.WaitGroup{}

	start := func(name appdef.QName) {
		startWG.Add(1)
		go func() {
			ctx, cancel := context.WithCancel(vvmCtx)
			rt := newActualizerRT(cancel)

			pa.mx.Lock()
			pa.actualizers[name] = rt
			pa.mx.Unlock()

			go func() {
				rt.state.Store(1) // started

				run(ctx, pa.app, pa.part, name)

				pa.mx.Lock()
				delete(pa.actualizers, name)
				pa.mx.Unlock()

				rt.state.Store(-1) // finished
			}()

			for rt.state.Load() == 0 {
				time.Sleep(time.Nanosecond) // wait until actualizer go-routine is started
			}
			startWG.Done()
		}()
	}

	pa.mx.RLock()
	appDef.Projectors(func(prj appdef.IProjector) {
		if !prj.Sync() {
			name := prj.QName()
			if _, exists := pa.actualizers[name]; !exists {
				start(name)
			}
		}
	})
	pa.mx.RUnlock()
	startWG.Wait() // wait for all new actualizers to start
}

// returns all deployed actualizers
func (pa *Actualizers) Enum() appdef.QNames {
	pa.mx.RLock()
	defer pa.mx.RUnlock()

	return appdef.QNamesFromMap(pa.actualizers)
}

type actualizerRT struct {
	cancel context.CancelFunc
	state  atomic.Int32 // 0: newly; +1: started; -1: finished
}

func newActualizerRT(cancel context.CancelFunc) *actualizerRT {
	return &actualizerRT{cancel: cancel}
}
