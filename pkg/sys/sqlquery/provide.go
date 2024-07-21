/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sqlquery

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(sprb istructsmem.IStatelessPkgResourcesBuilder, asp istructs.IAppStructsProvider) {
	sprb.AddFunc(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "SqlQuery"),
		provideEexecQrySqlQuery(asp),
	))
}
