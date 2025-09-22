/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schedulers

import (
	"context"
	"slices"
	"sync"

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

	ps.stopOlds(appDef)
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

// start actualizer
func (ps *PartitionSchedulers) start(vvmCtx context.Context, jws jWS, run Run) {
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
	<-done // wait until scheduler is started
}

// start new schedulers
func (ps *PartitionSchedulers) startNews(vvmCtx context.Context, appDef appdef.IAppDef, run Run) {
	for job := range appdef.Jobs(appDef.Types()) {
		name := job.QName()
		for wsID, wsNum := range ps.wsNumbers {
			jws := jWS{name, wsID, wsNum}
			if _, exists := ps.rt.Load(jws); !exists {
				ps.start(vvmCtx, jws, run)
			}
		}
	}
}

// stop old schedulers
func (ps *PartitionSchedulers) stopOlds(appDef appdef.IAppDef) {
	olds := make([]*runtime, 0)
	for jws, rt := range ps.rt.Range {
		// TODO: compare if job properties changed (cron, states, intents, etc.)
		name := jws.(jWS).QName
		if appdef.Job(appDef.Type, name) == nil {
			olds = append(olds, rt.(*runtime))
		}
	}

	wg := sync.WaitGroup{}
	for _, rt := range olds {
		wg.Add(1)
		go func() {
			rt.cancel()
			<-rt.done // wait until scheduler is finished
			wg.Done()
		}()
	}
	// wrong to watch over vvmCtx. See https://github.com/voedger/voedger/issues/3971
	wg.Wait() // wait for all old actualizers to stop
}

type jWS struct {
	appdef.QName // job name
	istructs.WSID
	istructs.AppWorkspaceNumber
}

type runtime struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func newRuntime(cancel context.CancelFunc) *runtime {
	return &runtime{
		cancel: cancel,
		done:   make(chan struct{}),
	}
}
