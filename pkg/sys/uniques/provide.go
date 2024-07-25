/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package uniques

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(sr istructsmem.IStatelessResources) {
	sr.AddProjectors(appdef.SysPackagePath, istructs.Projector{
		Name: qNameApplyUniques,
		Func: applyUniques,
	})
}

func ProvideEventValidator(cfg *istructsmem.AppConfigType) {
	cfg.AddEventValidators(eventUniqueValidator)
}