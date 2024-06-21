/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type nullActualizers struct{}

func (nullActualizers) DeployPartition(appdef.AppQName, istructs.PartitionID) error { return nil }
func (nullActualizers) UndeployPartition(appdef.AppQName, istructs.PartitionID)     {}
