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
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "DescribePackageNamesResult")).
			AddField("Names", appdef.DataKind_string, true).(appdef.IDef).QName(),
		provideQryDescribePackageNames(asp, cfg.Name),
	))
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "DescribePackage"),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "DescribePackageParams")).
			AddField(field_PackageName, appdef.DataKind_string, true).(appdef.IDef).QName(),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "DescribePackageResult")).
			AddField("PackageDesc", appdef.DataKind_string, true).(appdef.IDef).QName(),
		provideQryDescribePackage(asp, cfg.Name),
	))
}
