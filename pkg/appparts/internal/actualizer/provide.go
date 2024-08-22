/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package actualizer

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func New(app appdef.AppQName, part istructs.PartitionID) *Actualizers {
	return &Actualizers{
		app:         app,
		part:        part,
		actualizers: make(map[appdef.QName]*actualizerRT),
	}
}
