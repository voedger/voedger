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
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "DescribePackageNamesResult"), appdef.DefKind_Object).
			AddField("Names", appdef.DataKind_string, true).QName(),
		provideQryDescribePackageNames(asp, cfg.Name),
	))
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "DescribePackage"),
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "DescribePackageParams"), appdef.DefKind_Object).
			AddField(field_PackageName, appdef.DataKind_string, true).QName(),
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "DescribePackageResult"), appdef.DefKind_Object).
			AddField("PackageDesc", appdef.DataKind_string, true).QName(),
		provideQryDescribePackage(asp, cfg.Name),
	))
}
