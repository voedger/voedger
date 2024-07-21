/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package builtin

import (
	"runtime/debug"

	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func Provide(sprb istructsmem.IStatelessPkgResourcesBuilder, buildInfo *debug.BuildInfo, asp istorage.IAppStorageProvider) {
	sprb.AddFunc(istructsmem.NewCommandFunction(istructs.QNameCommandCUD, istructsmem.NullCommandExec))

	// Deprecated: use c.sys.CUD instead. Kept for backward compatibility only
	// to import via ImportBO
	sprb.AddFunc(istructsmem.NewCommandFunction(QNameCommandInit, istructsmem.NullCommandExec))

	provideRefIntegrityValidation(sprb)
	provideQryModules(sprb, buildInfo)

	provideQryEcho(sprb)
	provideQryGRCount(sprb)
	proivideRenameQName(sprb, asp)
	provideSysIsActiveValidation(sprb)
}
