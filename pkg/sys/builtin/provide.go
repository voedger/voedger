/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package builtin

import (
	"runtime/debug"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(cfg *istructsmem.AppConfigType, adf appdef.IAppDefBuilder, buildInfo *debug.BuildInfo, asp istorage.IAppStorageProvider,
	ep extensionpoints.IExtensionPoint) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(istructs.QNameCommandCUD, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))

	// Deprecated: use c.sys.CUD instead. Kept for backward compatibility only
	// to import via ImportBO
	cfg.Resources.Add(istructsmem.NewCommandFunction(QNameCommandInit, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))

	provideRefIntegrityValidation(cfg)
	provideQryModules(cfg, adf, buildInfo)

	provideQryEcho(cfg, adf, ep)
	provideQryGRCount(cfg, adf)
	proivideRenameQName(cfg, adf, asp)
}
