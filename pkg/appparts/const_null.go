/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
)

type nullActualizerRunner struct{}

func (nullActualizerRunner) NewAndRun(vvmCtx context.Context, _ appdef.AppQName, _ istructs.PartitionID, _ appdef.QName) {
	<-vvmCtx.Done()
}

func (nullActualizerRunner) SetAppPartitions(IAppPartitions) {}

type nullSchedulerRunner struct{}

func (nullSchedulerRunner) NewAndRun(vvmCtx context.Context, _ appdef.AppQName, _ istructs.PartitionID, _ istructs.AppWorkspaceNumber, _ istructs.WSID, _ appdef.QName) {
	<-vvmCtx.Done()
}

func (nullSchedulerRunner) SetAppPartitions(IAppPartitions) {}

func (nullSchedulerRunner) SchedulersTime() timeu.ITime {
	return timeu.NewITime()
}
