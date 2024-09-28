/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type nullActualizerRunner struct{}

func (nullActualizerRunner) NewAndRun(ctx context.Context, _ appdef.AppQName, _ istructs.PartitionID, _ appdef.QName) {
	<-ctx.Done()
}

func (nullActualizerRunner) SetAppPartitions(IAppPartitions) {}

type nullSchedulerRunner struct{}

func (nullSchedulerRunner) NewAndRun(ctx context.Context, _ appdef.AppQName, _ istructs.PartitionID, _ istructs.AppWorkspaceNumber, _ istructs.WSID, _ appdef.QName) {
	<-ctx.Done()
}

func (nullSchedulerRunner) SetAppPartitions(IAppPartitions) {}
