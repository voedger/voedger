/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schedulers

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// Run is a function that runs scheduler for the specified job.
type Run func(ctx context.Context, app appdef.AppQName, partID istructs.PartitionID, wsIdx istructs.AppWorkspaceNumber, wsID istructs.WSID, job appdef.QName)

// PartitionSchedulers manages schedulers deployment for the specified application partition.
type PartitionSchedulers struct {
	mx                    sync.RWMutex
	appQName              appdef.AppQName
	partitionID           istructs.PartitionID
	appWSNumbers          map[istructs.WSID]istructs.AppWorkspaceNumber
	jobsInAppWSIDRuntimes map[appdef.QName]map[istructs.WSID]*runtime
}

func newPartitionSchedulers(appQName appdef.AppQName, partCount istructs.NumAppPartitions, wsCount istructs.NumAppWorkspaces, partitionID istructs.PartitionID) *PartitionSchedulers {
	return &PartitionSchedulers{
		appQName:              appQName,
		partitionID:           partitionID,
		appWSNumbers:          AppWorkspacesHandledByPartition(partCount, wsCount, partitionID),
		jobsInAppWSIDRuntimes: make(map[appdef.QName]map[istructs.WSID]*runtime),
	}
}

// Deploys partition schedulers: stops schedulers for removed jobs and
// starts schedulers for new jobs using the specified run function.
func (ps *PartitionSchedulers) Deploy(vvmCtx context.Context, appDef appdef.IAppDef, run Run) {
	if len(ps.appWSNumbers) == 0 {
		return // no application workspaces handled by this partition
	}

	ps.stopOlds(vvmCtx, appDef)
	ps.startNews(vvmCtx, appDef, run)
}

// Returns all deployed schedulers.
//
// Returned map keys - job names, values - workspace IDs.
func (ps *PartitionSchedulers) Enum() map[appdef.QName][]istructs.WSID {
	ps.mx.RLock()
	defer ps.mx.RUnlock()

	res := make(map[appdef.QName][]istructs.WSID)
	for name, wsRT := range ps.jobsInAppWSIDRuntimes {
		ws := make([]istructs.WSID, 0, len(wsRT))
		for wsID := range wsRT {
			ws = append(ws, wsID)
		}
		slices.Sort(ws)
		res[name] = ws
	}
	return res
}

// Wait while all schedulers are finished.
//
// Contexts for schedulers should be stopped. Here we just wait for schedulers to finish
func (ps *PartitionSchedulers) Wait() {
	for {
		ps.mx.RLock()
		cnt := len(ps.jobsInAppWSIDRuntimes)
		ps.mx.RUnlock()
		if cnt == 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}
}

// Wait waits for all schedulers to finish.
// Returns true if all schedulers finished before the timeout.
// Returns false if the timeout is reached.
func (ps *PartitionSchedulers) WaitTimeout(timeout time.Duration) (finished bool) {
	done := make(chan struct{})
	go func() {
		ps.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// start actualizer
func (ps *PartitionSchedulers) start(vvmCtx context.Context, name appdef.QName, run Run, wg *sync.WaitGroup) {

	ps.mx.Lock()
	ps.jobsInAppWSIDRuntimes[name] = make(map[istructs.WSID]*runtime)
	ps.mx.Unlock()

	for appWSID, appWSNumber := range ps.appWSNumbers {
		ctx, cancel := context.WithCancel(vvmCtx)
		rt := newRuntime(cancel)

		ps.mx.Lock()
		ps.jobsInAppWSIDRuntimes[name][appWSID] = rt
		ps.mx.Unlock()

		done := make(chan struct{})
		go func(appWSNumber istructs.AppWorkspaceNumber, appWSID istructs.WSID) {
			close(done) // scheduler started

			run(ctx, ps.appQName, ps.partitionID, appWSNumber, appWSID, name)

			ps.mx.Lock()
			delete(ps.jobsInAppWSIDRuntimes[name], appWSID)
			if len(ps.jobsInAppWSIDRuntimes[name]) == 0 {
				delete(ps.jobsInAppWSIDRuntimes, name)
			}
			ps.mx.Unlock()

			close(rt.done) // scheduler finished
		}(appWSNumber, appWSID)

		select {
		case <-done: // wait until scheduler is started
		case <-vvmCtx.Done():
		}
	}

	wg.Done()
}

// start new schedulers
func (ps *PartitionSchedulers) startNews(vvmCtx context.Context, appDef appdef.IAppDef, run Run) {
	news := make(map[appdef.QName]struct{})
	ps.mx.RLock()
	for job := range appdef.Jobs(appDef.Types) {
		name := job.QName()
		if _, exists := ps.jobsInAppWSIDRuntimes[name]; !exists {
			news[name] = struct{}{}
		}
	}
	ps.mx.RUnlock()

	done := make(chan struct{})
	go func() {
		startWG := sync.WaitGroup{}
		for name := range news {
			startWG.Add(1)
			go ps.start(vvmCtx, name, run, &startWG)
		}
		startWG.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-vvmCtx.Done():
	}
}

// stop scheduler
func (ps *PartitionSchedulers) stop(vvmCtx context.Context, rt *runtime, wg *sync.WaitGroup) {
	rt.cancel()
	select {
	case <-rt.done: // wait until scheduler is finished
	case <-vvmCtx.Done():
	}
	wg.Done()
}

// stop old schedulers
func (ps *PartitionSchedulers) stopOlds(vvmCtx context.Context, appDef appdef.IAppDef) {
	ps.mx.RLock()
	olds := make([]*runtime, 0)
	for name, wsRT := range ps.jobsInAppWSIDRuntimes {
		// TODO: compare if job properties changed (cron, states, intents, etc.)
		if appdef.Job(appDef.Type, name) == nil {
			for _, rt := range wsRT {
				olds = append(olds, rt)
			}
		}
	}
	ps.mx.RUnlock()

	done := make(chan struct{})
	go func() {
		stopWG := sync.WaitGroup{}
		for _, rt := range olds {
			stopWG.Add(1)
			go ps.stop(vvmCtx, rt, &stopWG)
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
