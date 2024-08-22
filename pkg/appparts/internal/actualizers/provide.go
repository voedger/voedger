/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package actualizers

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func New(app appdef.AppQName, part istructs.PartitionID) *PartitionActualizers {
	return newActualizers(app, part)
}
