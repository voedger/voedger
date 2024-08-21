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

type nullProcessorRunner struct{}

func (nullProcessorRunner) NewAndRun(ctx context.Context, _ appdef.AppQName, _ istructs.PartitionID, _ appdef.QName) {
	<-ctx.Done()
}

func (nullProcessorRunner) SetAppPartitions(IAppPartitions) {}
