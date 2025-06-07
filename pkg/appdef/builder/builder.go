/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package builder

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/apps"
	"github.com/voedger/voedger/pkg/appdef/sys"
)

func New() appdef.IAppDefBuilder {
	app := apps.NewAppDef()
	adb := apps.NewAppDefBuilder(app)
	sys.MakeSysPackage(adb) // Initialize the system package
	return adb
}
