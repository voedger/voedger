/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package actualizers

import (
	"context"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// Run is a function that runs actualizer for the specified projector.
type Run func(context.Context, appdef.AppQName, istructs.PartitionID, appdef.QName)

// PartitionActualizers manages actualizers deployment for the specified application partition.
type PartitionActualizers struct {
	app  appdef.AppQName
	part istructs.PartitionID
	rt   sync.Map // appdef.QName -> *runtime
	rtWG sync.WaitGroup
}

func New(app appdef.AppQName, part istructs.PartitionID) *PartitionActualizers {
	return &PartitionActualizers{
		app:  app,
		part: part,
		rt:   sync.Map{},
		rtWG: sync.WaitGroup{},
	}
}

// Deploys partition actualizers: stops actualizers for removed projectors and
// starts actualizers for new projectors using the specified run function.
func (pa *PartitionActualizers) Deploy(vvmCtx context.Context, appDef appdef.IAppDef, run Run) {
	pa.stopOlds(vvmCtx, appDef)
	pa.startNews(vvmCtx, appDef, run)
}

// Returns all deployed actualizers
func (pa *PartitionActualizers) Enum() appdef.QNames {
	names := appdef.QNames{}
	for n := range pa.rt.Range {
		names.Add(n.(appdef.QName))
	}
	return names
}

// Wait waits for all actualizers to finish.
//
// The context should be stopped before calling this method. Here we just wait for actualizers to finish.
func (pa *PartitionActualizers) Wait() {
	pa.rtWG.Wait()
}

// Wait waits for all actualizers to finish.
// Returns true if all actualizers finished before the timeout.
// Returns false if the timeout is reached.
func (pa *PartitionActualizers) WaitTimeout(timeout time.Duration) (finished bool) {
	done := make(chan struct{})
	go func() {
		pa.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// async start actualizer
func (pa *PartitionActualizers) start(vvmCtx context.Context, name appdef.QName, run Run, wg *sync.WaitGroup) {
	ctx, cancel := context.WithCancel(vvmCtx)
	rt := newRuntime(cancel)

	pa.rt.Store(name, rt)

	pa.rtWG.Add(1)

	done := make(chan struct{})
	go func() {
		close(done) // actualizer started

		defer func() {
			pa.rt.Delete(name)
			close(rt.done) // actualizer finished
			pa.rtWG.Done()
		}()

		run(ctx, pa.app, pa.part, name)
	}()

	select {
	case <-done: // wait until actualizer is started
	case <-vvmCtx.Done():
	}

	wg.Done()
}

// async start new actualizers
func (pa *PartitionActualizers) startNews(vvmCtx context.Context, appDef appdef.IAppDef, run Run) {
	news := make(map[appdef.QName]struct{})
	for prj := range appdef.Projectors(appDef.Types()) {
		if !prj.Sync() {
			name := prj.QName()
			if _, exists := pa.rt.Load(name); !exists {
				news[name] = struct{}{}
			}
		}
	}

	done := make(chan struct{})
	go func() {
		startWG := sync.WaitGroup{}
		for name := range news {
			startWG.Add(1)
			go pa.start(vvmCtx, name, run, &startWG)
		}
		startWG.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-vvmCtx.Done():
	}
}

// async stop actualizer
func (pa *PartitionActualizers) stop(vvmCtx context.Context, rt *runtime, wg *sync.WaitGroup) {
	rt.cancel()
	select {
	case <-rt.done: // wait until actualizer is finished
	case <-vvmCtx.Done():
	}
	wg.Done()
}

// async stop old actualizers
func (pa *PartitionActualizers) stopOlds(vvmCtx context.Context, appDef appdef.IAppDef) {
	olds := make([]*runtime, 0)
	for name, rt := range pa.rt.Range {
		name := name.(appdef.QName)
		rt := rt.(*runtime)
		// TODO: compare if projector properties changed (events, sync/async, etc.)
		if appdef.Projector(appDef.Type, name) == nil {
			olds = append(olds, rt)
		}
	}

	done := make(chan struct{})
	go func() {
		stopWG := sync.WaitGroup{}
		for _, rt := range olds {
			stopWG.Add(1)
			go pa.stop(vvmCtx, rt, &stopWG)
		}
		stopWG.Wait() // wait for all old actualizers to stop
		close(done)
	}()

	select {
	case <-done:
	case <-vvmCtx.Done():
	}
}

type runtime struct {
	cancel context.CancelFunc
	done   chan []struct{}
}

func newRuntime(cancel context.CancelFunc) *runtime {
	return &runtime{
		cancel: cancel,
		done:   make(chan []struct{}),
	}
}
