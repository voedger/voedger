/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package authnz

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
)

func Provide(appDefBuilder appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) {
	apps.Parse(schemasFS, appdef.SysPackage, ep)
}
