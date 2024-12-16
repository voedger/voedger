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
	appQName    appdef.AppQName
	partitionID istructs.PartitionID
	wsNumbers   map[istructs.WSID]istructs.AppWorkspaceNumber
	rt          sync.Map // {appdef.QName, istructs.WSID} -> *runtime
	rtWG        sync.WaitGroup
}

func New(app appdef.AppQName, partCount istructs.NumAppPartitions, wsCount istructs.NumAppWorkspaces, part istructs.PartitionID) *PartitionSchedulers {
	return &PartitionSchedulers{
		appQName:    app,
		partitionID: part,
		wsNumbers:   AppWorkspacesHandledByPartition(partCount, wsCount, part),
		rt:          sync.Map{},
		rtWG:        sync.WaitGroup{},
	}
}

// Deploys partition schedulers: stops schedulers for removed jobs and
// starts schedulers for new jobs using the specified run function.
func (ps *PartitionSchedulers) Deploy(vvmCtx context.Context, appDef appdef.IAppDef, run Run) {
	if len(ps.wsNumbers) == 0 {
		return // no application workspaces handled by this partition
	}

	ps.stopOlds(vvmCtx, appDef)
	ps.startNews(vvmCtx, appDef, run)
}

// Returns all deployed schedulers.
//
// Returned map keys - job names, values - workspace IDs.
func (ps *PartitionSchedulers) Enum() map[appdef.QName][]istructs.WSID {
	res := make(map[appdef.QName][]istructs.WSID)
	for jws := range ps.rt.Range {
		jws := jws.(jWS)
		if _, exists := res[jws.QName]; !exists {
			res[jws.QName] = make([]istructs.WSID, 0, len(ps.wsNumbers))
		}
		res[jws.QName] = append(res[jws.QName], jws.WSID)
	}
	for n, wsIDs := range res {
		slices.Sort(wsIDs)
		res[n] = wsIDs
	}
	return res
}

// Wait while all schedulers are finished.
//
// Contexts for schedulers should be stopped. Here we just wait for schedulers to finish
func (ps *PartitionSchedulers) Wait() {
	ps.rtWG.Wait()
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
func (ps *PartitionSchedulers) start(vvmCtx context.Context, jws jWS, run Run, wg *sync.WaitGroup) {
	ctx, cancel := context.WithCancel(vvmCtx)
	rt := newRuntime(cancel)

	ps.rt.Store(jws, rt)

	ps.rtWG.Add(1)

	done := make(chan struct{})
	go func() {
		close(done) // scheduler started

		defer func() {
			ps.rt.Delete(jws)
			close(rt.done) // scheduler finished
			ps.rtWG.Done()
		}()

		run(ctx, ps.appQName, ps.partitionID, jws.AppWorkspaceNumber, jws.WSID, jws.QName)
	}()

	select {
	case <-done: // wait until scheduler is started
	case <-vvmCtx.Done():
	}

	wg.Done()
}

// start new schedulers
func (ps *PartitionSchedulers) startNews(vvmCtx context.Context, appDef appdef.IAppDef, run Run) {
	news := make(map[jWS]struct{})
	for job := range appdef.Jobs(appDef.Types()) {
		name := job.QName()
		for wsID, wsNum := range ps.wsNumbers {
			jws := jWS{name, wsID, wsNum}
			if _, exists := ps.rt.Load(jws); !exists {
				news[jws] = struct{}{}
			}
		}
	}

	done := make(chan struct{})
	go func() {
		startWG := sync.WaitGroup{}
		for jws := range news {
			startWG.Add(1)
			go ps.start(vvmCtx, jws, run, &startWG)
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
	olds := make([]*runtime, 0)
	for jws, rt := range ps.rt.Range {
		// TODO: compare if job properties changed (cron, states, intents, etc.)
		name := jws.(jWS).QName
		if appdef.Job(appDef.Type, name) == nil {
			olds = append(olds, rt.(*runtime))
		}
	}

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

type jWS struct {
	appdef.QName // job name
	istructs.WSID
	istructs.AppWorkspaceNumber
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
