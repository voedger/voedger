/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schedulers

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// Run is a function that runs scheduler for the specified job.
type Run func(ctx context.Context, app appdef.AppQName, partID istructs.PartitionID, wsIdx int, wsID istructs.WSID, job appdef.QName)

// PartitionSchedulers manages schedulers deployment for the specified application partition.
type PartitionSchedulers struct {
	mx   sync.RWMutex
	app  appdef.AppQName
	part istructs.PartitionID
	ws   map[istructs.WSID]int
	rt   map[appdef.QName]map[istructs.WSID]*runtime
}

func newPartitionSchedulers(app appdef.AppQName, partCount istructs.NumAppPartitions, wsCount istructs.NumAppWorkspaces, part istructs.PartitionID) *PartitionSchedulers {
	return &PartitionSchedulers{
		app:  app,
		part: part,
		ws:   AppWorkspacesHandledByPartition(partCount, wsCount, part),
		rt:   make(map[appdef.QName]map[istructs.WSID]*runtime),
	}
}

// Deploys partition schedulers: stops schedulers for removed jobs and
// starts schedulers for new jobs using the specified run function.
func (ps *PartitionSchedulers) Deploy(vvmCtx context.Context, appDef appdef.IAppDef, run Run) {
	if len(ps.ws) == 0 {
		return // no application workspaces handled by this partition
	}

	// async stop old actualizers
	stopWG := sync.WaitGroup{}
	ps.mx.RLock()
	for name, wsRT := range ps.rt {
		// TODO: compare if job properties changed (cron, etc.)
		if appDef.Job(name) == nil {
			for _, rt := range wsRT {
				stopWG.Add(1)
				go func(rt *runtime) {
					rt.cancel()
					for rt.state.Load() >= 0 {
						time.Sleep(time.Nanosecond) // wait until scheduler is finished
					}
					stopWG.Done()
				}(rt)
			}
		}
	}
	ps.mx.RUnlock()
	stopWG.Wait() // wait for all old schedulers to stop

	// async start new schedulers
	startWG := sync.WaitGroup{}

	start := func(name appdef.QName) {
		startWG.Add(1)
		go func() {
			ps.mx.Lock()
			ps.rt[name] = make(map[istructs.WSID]*runtime)
			ps.mx.Unlock()

			for wsID, wsIdx := range ps.ws {
				ctx, cancel := context.WithCancel(vvmCtx)
				rt := newRuntime(cancel)

				ps.mx.Lock()
				ps.rt[name][wsID] = rt
				ps.mx.Unlock()

				go func(wsIdx int, wsID istructs.WSID) {
					rt.state.Store(1) // started

					run(ctx, ps.app, ps.part, wsIdx, wsID, name)

					ps.mx.Lock()
					delete(ps.rt[name], wsID)
					if len(ps.rt[name]) == 0 {
						delete(ps.rt, name)
					}
					ps.mx.Unlock()

					rt.state.Store(-1) // finished
				}(wsIdx, wsID)

				for rt.state.Load() == 0 {
					time.Sleep(time.Nanosecond) // wait until actualizer go-routine is started
				}
			}
			startWG.Done()
		}()
	}

	ps.mx.RLock()
	appDef.Jobs(func(job appdef.IJob) {
		name := job.QName()
		if _, exists := ps.rt[name]; !exists {
			start(name)
		}
	})
	ps.mx.RUnlock()
	startWG.Wait() // wait for all new schedulers to start
}

// Returns all deployed schedulers.
//
// Returned map keys - job names, values - workspace IDs.
func (ps *PartitionSchedulers) Enum() map[appdef.QName][]istructs.WSID {
	ps.mx.RLock()
	defer ps.mx.RUnlock()

	res := make(map[appdef.QName][]istructs.WSID)
	for name, wsRT := range ps.rt {
		ws := make([]istructs.WSID, 0, len(wsRT))
		for wsID := range wsRT {
			ws = append(ws, wsID)
		}
		slices.Sort(ws)
		res[name] = ws
	}
	return res
}

type runtime struct {
	state  atomic.Int32 // 0: newly; +1: started; -1: finished
	cancel context.CancelFunc
}

func newRuntime(cancel context.CancelFunc) *runtime {
	return &runtime{cancel: cancel}
}
