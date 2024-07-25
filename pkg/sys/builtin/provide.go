/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package builtin

import (
	"runtime/debug"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(sr istructsmem.IStatelessResources, buildInfo *debug.BuildInfo, asp istorage.IAppStorageProvider) {
	sr.AddCommands(appdef.SysPackagePath,
		istructsmem.NewCommandFunction(istructs.QNameCommandCUD, istructsmem.NullCommandExec),

		// Deprecated: use c.sys.CUD instead. Kept for backward compatibility only
		// to import via ImportBO
		istructsmem.NewCommandFunction(QNameCommandInit, istructsmem.NullCommandExec),
	)

	provideRefIntegrityValidation(sr)
	provideQryModules(sr, buildInfo)

	provideQryEcho(sr)
	provideQryGRCount(sr)
	proivideRenameQName(sr, asp)
}

func ProvideCUDValidators(cfg *istructsmem.AppConfigType) {
	cfg.AddCUDValidators(provideRefIntegrityValidator())
}
