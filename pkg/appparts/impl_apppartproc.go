/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
)

type (
	partitionProcessors struct {
		mx   sync.RWMutex
		part *appPartitionRT
		proc map[appdef.QName]*procRT
	}

	procRT struct {
		kind   ProcessorKind // actualizer or scheduler
		cancel context.CancelFunc
		state  atomic.Int32 // 0: newly; +1: started; -1: finished
	}
)

func newPartitionProcessors(part *appPartitionRT) *partitionProcessors {
	return &partitionProcessors{
		part: part,
		proc: map[appdef.QName]*procRT{},
	}
}

func newProcRT(kind ProcessorKind, cancel context.CancelFunc) *procRT {
	return &procRT{kind: kind, cancel: cancel}
}

// deploys partition processors (actualizers and schedulers):
//   - stops actualizers for removed projectors and starts actualizers for new projectors
//   - stops schedulers for removed jobs and starts schedulers for new jobs.
func (pp *partitionProcessors) deploy() {
	appDef := pp.part.app.lastestVersion.appDef()

	// async stop old processors
	stopWG := sync.WaitGroup{}
	pp.mx.RLock()
	for name, rt := range pp.proc {
		old := false
		switch rt.kind {
		case ProcessorKind_Actualizer:
			// TODO: compare if projector properties changed (events, sync/async, etc.)
			old = appDef.Projector(name) == nil
		case ProcessorKind_Scheduler:
			// TODO: compare if scheduler properties changed (cron, etc.)
			old = appDef.Job(name) == nil
		}
		if old {
			stopWG.Add(1)
			go func(rt *procRT) {
				rt.cancel()
				for rt.state.Load() >= 0 {
					time.Sleep(time.Nanosecond) // wait until processor is finished
				}
				stopWG.Done()
			}(rt)
		}
	}
	pp.mx.RUnlock()
	stopWG.Wait() // wait for all old processors to stop

	// async start new processors
	startWG := sync.WaitGroup{}

	start := func(name appdef.QName, kind ProcessorKind) {
		startWG.Add(1)
		go func() {
			ctx, cancel := context.WithCancel(pp.part.app.apps.vvmCtx)
			rt := newProcRT(kind, cancel)

			pp.mx.Lock()
			pp.proc[name] = rt
			pp.mx.Unlock()

			go func() {
				rt.state.Store(1) // started

				pp.part.app.apps.processors[kind].NewAndRun(
					ctx,
					pp.part.app.name,
					pp.part.id,
					name,
				)

				pp.mx.Lock()
				delete(pp.proc, name)
				pp.mx.Unlock()

				rt.state.Store(-1) // finished
			}()

			for rt.state.Load() == 0 {
				time.Sleep(time.Nanosecond) // wait until processor is started
			}
			startWG.Done()
		}()
	}

	pp.mx.RLock()
	appDef.Projectors(func(prj appdef.IProjector) {
		if !prj.Sync() {
			name := prj.QName()
			if _, exists := pp.proc[name]; !exists {
				start(name, ProcessorKind_Actualizer)
			}
		}
	})
	appDef.Jobs(func(job appdef.IJob) {
		name := job.QName()
		if _, exists := pp.proc[name]; !exists {
			start(name, ProcessorKind_Scheduler)
		}
	})
	pp.mx.RUnlock()
	startWG.Wait() // wait for all new processors to start
}

// returns all deployed processors
func (pp *partitionProcessors) enum() appdef.QNames {
	pp.mx.RLock()
	defer pp.mx.RUnlock()

	return appdef.QNamesFromMap(pp.proc)
}
