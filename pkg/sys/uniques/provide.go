/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package uniques

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(cfg *istructsmem.AppConfigType) {
	cfg.AddSyncProjectors(istructs.Projector{
		Name: qNameApplyUniques,
		Func: provideApplyUniques(cfg.AppDef),
	})
	cfg.AddEventValidators(eventUniqueValidator)
}
