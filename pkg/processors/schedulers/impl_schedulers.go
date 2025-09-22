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
	retrier "github.com/voedger/voedger/pkg/goutils/retry"
	"github.com/voedger/voedger/pkg/istructs"
)

/*
implements:

	appparts.ISchedulerRunner
*/
type schedulers struct {
	cfg      BasicSchedulerConfig
	wait     sync.WaitGroup
	appParts appparts.IAppPartitions
}

func newSchedulers(cfg BasicSchedulerConfig) appparts.ISchedulerRunner {
	return &schedulers{
		cfg: cfg,
	}
}

// Creates and runs new actualizer for specified partition.
//
// # apparts.IActualizerRunner.NewAndRun
func (a *schedulers) NewAndRun(ctx context.Context, app appdef.AppQName, partition istructs.PartitionID, appWSIdx istructs.AppWorkspaceNumber, wsid istructs.WSID, job appdef.QName) {
	act := &scheduler{
		job: job,
		conf: SchedulerConfig{
			BasicSchedulerConfig: a.cfg,
			AppQName:             app,
			Partition:            partition,
			Workspace:            wsid,
			AppWSIdx:             appWSIdx,
		},
		appParts:   a.appParts,
		retrierCfg: retrier.NewConfig(schedulerRetryDelay, schedulerRetryDelay),
	}
	act.Prepare()

	a.wait.Add(1)
	act.Run(ctx)
	a.wait.Done()
}

func (a *schedulers) SetAppPartitions(ap appparts.IAppPartitions) {
	a.appParts = ap
}
