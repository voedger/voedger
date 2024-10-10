/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package actualizers

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// Run is a function that runs actualizer for the specified projector.
type Run func(context.Context, appdef.AppQName, istructs.PartitionID, appdef.QName)

// PartitionActualizers manages actualizers deployment for the specified application partition.
type PartitionActualizers struct {
	mx   sync.RWMutex
	app  appdef.AppQName
	part istructs.PartitionID
	rt   map[appdef.QName]*runtime
}

func newActualizers(app appdef.AppQName, part istructs.PartitionID) *PartitionActualizers {
	return &PartitionActualizers{
		app:  app,
		part: part,
		rt:   make(map[appdef.QName]*runtime),
	}
}

// Deploys partition actualizers: stops actualizers for removed projectors and
// starts actualizers for new projectors using the specified run function.
func (pa *PartitionActualizers) Deploy(vvmCtx context.Context, appDef appdef.IAppDef, run Run) {
	// async stop old actualizers
	stopWG := sync.WaitGroup{}
	pa.mx.RLock()
	for name, rt := range pa.rt {
		// TODO: compare if projector properties changed (events, sync/async, etc.)
		if appDef.Projector(name) == nil {
			stopWG.Add(1)
			go func(rt *runtime) {
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
			rt := newRuntime(cancel)

			pa.mx.Lock()
			pa.rt[name] = rt
			pa.mx.Unlock()

			go func() {
				rt.state.Store(1) // started

				run(ctx, pa.app, pa.part, name)

				pa.mx.Lock()
				delete(pa.rt, name)
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
	for prj := range appDef.Projectors {
		if !prj.Sync() {
			name := prj.QName()
			if _, exists := pa.rt[name]; !exists {
				start(name)
			}
		}
	}
	pa.mx.RUnlock()
	startWG.Wait() // wait for all new actualizers to start
}

// Returns all deployed actualizers
func (pa *PartitionActualizers) Enum() appdef.QNames {
	pa.mx.RLock()
	defer pa.mx.RUnlock()

	return appdef.QNamesFromMap(pa.rt)
}

// Wait waits for all actualizers to finish.
//
// The context should be stopped before calling this method. Here we just wait for actualizers to finish.
func (pa *PartitionActualizers) Wait() {
	for {
		pa.mx.RLock()
		cnt := len(pa.rt)
		pa.mx.RUnlock()
		if cnt == 0 {
			break
		}
		time.Sleep(time.Nanosecond)
	}
}

type runtime struct {
	state  atomic.Int32 // 0: newly; +1: started; -1: finished
	cancel context.CancelFunc
}

func newRuntime(cancel context.CancelFunc) *runtime {
	return &runtime{cancel: cancel}
}
