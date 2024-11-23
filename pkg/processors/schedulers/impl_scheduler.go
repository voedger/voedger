/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package schedulers

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/state/stateprovide"
)

type scheduler struct {
	name string
	conf SchedulerConfig
	job  appdef.QName
	// init:
	jobInErrAddr *imetrics.MetricValue
	schedule     cron.Schedule
	// run:
	ctx          context.Context
	projErrState int32 // 0 - no error, 1 - error
	appParts     appparts.IAppPartitions
}

func (a *scheduler) Prepare() {
	if a.conf.IntentsLimit == 0 {
		a.conf.IntentsLimit = defaultIntentsLimit
	}

	if a.conf.AfterError == nil {
		a.conf.AfterError = time.After
	}

	if a.conf.LogError == nil {
		a.conf.LogError = logger.Error
	}
	a.name = fmt.Sprintf("%v [idx: %d, id: %d]", a.job, a.conf.AppWSIdx, a.conf.Workspace)
}

func (a *scheduler) Run(ctx context.Context) {
	var err error
	if err = a.waitForAppDeploy(ctx); err != nil {
		panic(err)
	}
	for ctx.Err() == nil {
		a.ctx = ctx
		if err = a.init(ctx); err == nil {
			a.keepRunning()
		}
		a.finit() // even execute if a.init has failed
		if ctx.Err() == nil && err != nil {
			a.conf.LogError(a.name, err)
			select {
			case <-ctx.Done():
			case <-a.conf.AfterError(schedulerErrorDelay):
			}
		}
	}
}

func (a *scheduler) runJob() {
	var err error
	var borrowedPartition appparts.IAppPartition
	defer func() {
		if borrowedPartition != nil {
			borrowedPartition.Release()
		}
		if err != nil {
			a.conf.LogError(a.name, err)
			if atomic.CompareAndSwapInt32(&a.projErrState, 0, 1) {
				if a.jobInErrAddr != nil {
					a.jobInErrAddr.Increase(1)
				}
			}
		}
	}()
	borrowedPartition, err = a.appParts.WaitForBorrow(a.ctx, a.conf.AppQName, a.conf.Partition, appparts.ProcessorKind_Scheduler)
	if err != nil {
		return
	}
	state := stateprovide.ProvideSchedulerStateFactory()(
		a.ctx,
		func() istructs.IAppStructs { return borrowedPartition.AppStructs() },
		func() istructs.WSID { return a.conf.Workspace },
		func(view appdef.QName, wsid istructs.WSID, offset istructs.Offset) {
			a.conf.Broker.Update(in10n.ProjectionKey{
				App:        a.conf.AppQName,
				Projection: view,
				WS:         wsid,
			}, offset)
		},
		a.conf.SecretReader,
		a.conf.Tokens,
		a.conf.Federation,
		func() int64 { return a.conf.Time.Now().Unix() },
		a.conf.IntentsLimit,
		a.conf.Opts...)

	if err = borrowedPartition.Invoke(a.ctx, a.job, state, state); err != nil {
		return
	}
	if logger.IsVerbose() {
		logger.Verbose("invoked " + a.name)
	}
	err = state.ApplyIntents()
	if err != nil {
		return
	}
	if err == nil && a.jobInErrAddr != nil {
		if atomic.CompareAndSwapInt32(&a.projErrState, 1, 0) {
			a.jobInErrAddr.Increase(-1)
		}
	}
}

func (a *scheduler) waitForAppDeploy(ctx context.Context) error {
	start := time.Now()
	for ctx.Err() == nil {
		ap, err := a.appParts.Borrow(a.conf.AppQName, a.conf.Partition, appparts.ProcessorKind_Actualizer)
		if err == nil || errors.Is(err, appparts.ErrNotAvailableEngines) {
			if ap != nil {
				ap.Release()
			}
			return nil
		}
		if !errors.Is(err, appparts.ErrNotFound) {
			return err
		}
		if time.Since(start) >= initFailureErrorLogInterval {
			logger.Error(fmt.Sprintf("app %s part %d actualizer %s: failed to init in 30 seconds", a.conf.AppQName, a.conf.Partition, a.name))
			start = time.Now()
		}
		time.Sleep(borrowRetryDelay)
	}
	return nil // consider "context canceled" as expected error
}

func (a *scheduler) init(_ context.Context) (err error) {

	appDef, err := a.appParts.AppDef(a.conf.AppQName)
	if err != nil {
		return err
	}
	jobType := appdef.Job(appDef.Type, a.job)
	if jobType == nil {
		return fmt.Errorf("job %s is not defined in AppDef", a.job)
	}

	if a.conf.Metrics != nil {
		a.jobInErrAddr = a.conf.Metrics.AppMetricAddr(JobsInError, string(a.conf.VvmName), a.conf.AppQName)
	}

	parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	a.schedule, err = parser.Parse(jobType.CronSchedule())
	if err != nil {
		return fmt.Errorf("failed to parse cron schedule: %w", err)
	}
	logger.Trace(a.name, "started")
	return nil
}

func (a *scheduler) keepRunning() {
	now := a.conf.Time.Now()
	nextTime := a.schedule.Next(now)
	for a.ctx.Err() == nil {
		logger.Info(a.name, "schedule", "now", now, "next", nextTime)
		timerChan := a.conf.Time.NewTimerChan(nextTime.Sub(now))
		select {
		case <-a.ctx.Done():
			return
		case now = <-timerChan:
			logger.Info(a.name, "wake", "now", now)
			a.runJob()
			nextTime = a.schedule.Next(now)
		}
	}
}

func (a *scheduler) finit() {
	if logger.IsTrace() {
		logger.Trace(a.name + "s finalized")
	}
	if a.jobInErrAddr != nil {
		if atomic.CompareAndSwapInt32(&a.projErrState, 1, 0) {
			a.jobInErrAddr.Increase(-1)
		}
	}
}
