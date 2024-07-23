/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package describe

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(sr istructsmem.IStatelessResources) {
	sr.AddQueries(appdef.SysPackagePath,
		istructsmem.NewQueryFunction(
			appdef.NewQName(appdef.SysPackage, "DescribePackageNames"),
			qryDescribePackageNames,
		),
		istructsmem.NewQueryFunction(
			appdef.NewQName(appdef.SysPackage, "DescribePackage"),
			qryDescribePackage,
		),
	)
}
