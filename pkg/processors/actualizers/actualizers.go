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
	"github.com/voedger/voedger/pkg/appparts"
	retrier "github.com/voedger/voedger/pkg/goutils/retry"
	"github.com/voedger/voedger/pkg/istructs"
)

// actualizers is a set of actualizers for application partitions.
//
// # Implements:
//
//	appparts.IActualizerRunner
type actualizers struct {
	cfg      BasicAsyncActualizerConfig
	wait     sync.WaitGroup
	appParts appparts.IAppPartitions
}

func newActualizers(cfg BasicAsyncActualizerConfig) *actualizers {
	return &actualizers{
		cfg:  cfg,
		wait: sync.WaitGroup{},
	}
}

// Creates and runs new actualizer for specified partition.
//
// # apparts.IActualizerRunner.NewAndRun
func (a *actualizers) NewAndRun(ctx context.Context, app appdef.AppQName, part istructs.PartitionID, prj appdef.QName) {
	act := &asyncActualizer{
		projectorQName: prj,
		conf: AsyncActualizerConf{
			BasicAsyncActualizerConfig: a.cfg,
			AppQName:                   app,
			PartitionID:                part,
		},
		appParts:   a.appParts,
		retrierCfg: retrier.NewConfig(time.Duration(a.cfg.RetryDelay), time.Duration(a.cfg.RetryDelay)),
	}
	act.Prepare()
	a.wait.Add(1)
	act.Run(ctx)
	a.wait.Done()
}

func (a *actualizers) SetAppPartitions(ap appparts.IAppPartitions) {
	a.appParts = ap
}
