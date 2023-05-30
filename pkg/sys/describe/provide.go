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
	res := appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "DescribePackageNamesResult"))
	res.AddField("Names", appdef.DataKind_string, true)
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "DescribePackageNames"), appdef.NullQName, res.QName(),
		provideQryDescribePackageNames(asp, cfg.Name),
	))

	pars := appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "DescribePackageParams"))
	pars.AddField(field_PackageName, appdef.DataKind_string, true)
	res = appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "DescribePackageResult"))
	res.AddField("PackageDesc", appdef.DataKind_string, true)
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "DescribePackage"), pars.QName(), res.QName(),
		provideQryDescribePackage(asp, cfg.Name),
	))
}
