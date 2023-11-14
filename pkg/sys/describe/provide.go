/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package describe

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(cfg *istructsmem.AppConfigType, asp istructs.IAppStructsProvider, appDefBuilder appdef.IAppDefBuilder) {
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "DescribePackageNames"),
		appdef.NullQName,
		appdef.NullQName,
		provideQryDescribePackageNames(asp, cfg.Name),
	))
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "DescribePackage"),
		appdef.NullQName,
		appdef.NullQName,
		provideQryDescribePackage(asp, cfg.Name),
	))
}
