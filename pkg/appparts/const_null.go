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

func (nullProcessorRunner) NewAndRun(context.Context, appdef.AppQName, istructs.PartitionID, appdef.QName) {
}

func (nullProcessorRunner) SetAppPartitions(IAppPartitions) {}
