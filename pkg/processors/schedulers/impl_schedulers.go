/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package schedulers

import (
	"context"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/istructs"
)

/*
implements:

	pipeline.IServiceEx
	appparts.IProcessorRunner
*/
type schedulers struct {
	cfg  BasicSchedulerConfig
	wait sync.WaitGroup
}

func newSchedulers(cfg BasicSchedulerConfig) ISchedulersService {
	return &schedulers{
		cfg: cfg,
	}
}

// Creates and runs new actualizer for specified partition.
//
// # apparts.IActualizerRunner.NewAndRun
func (a *schedulers) NewAndRun(ctx context.Context, app appdef.AppQName, partition istructs.PartitionID, wsIdx int, wsid istructs.WSID, job appdef.QName) {
	act := &scheduler{
		job: job,
		conf: SchedulerConfig{
			BasicSchedulerConfig: a.cfg,
			AppQName:             app,
			Partition:            partition,
			Workspace:            wsid,
			WSIdx:                wsIdx,
		},
	}
	act.Prepare()

	a.wait.Add(1)
	act.Run(ctx)
	a.wait.Done()
}

// # pipeline.IService.Prepare
func (*schedulers) Prepare(interface{}) error { return nil }

// # pipeline.IService.Run
func (*schedulers) Run(context.Context) {
	panic("not implemented")
}

// # pipeline.IServiceEx.RunEx
func (a *schedulers) RunEx(_ context.Context, started func()) {
	started()
}

func (a *schedulers) SetAppPartitions(ap appparts.IAppPartitions) {
	a.cfg.AppPartitions = ap
}

func (a *schedulers) Stop() {
	// Cancellation has already been sent to the context by caller.
	// Here we are just waiting while all async actualizers are stopped
	a.wait.Wait()
}
