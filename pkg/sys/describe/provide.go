/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package describe

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(cfg *istructsmem.AppConfigType) {
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "DescribePackageNames"),
		qryDescribePackageNames,
	))
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "DescribePackage"),
		qryDescribePackage,
	))
}
