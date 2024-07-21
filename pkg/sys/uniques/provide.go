/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package uniques

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(sprb istructsmem.IStatelessPkgResourcesBuilder) {
	sprb.AddSyncProjectors(istructs.Projector{
		Name: qNameApplyUniques,
		Func: applyUniques,
	})
	sprb.AddEventValidators(eventUniqueValidator)
}
