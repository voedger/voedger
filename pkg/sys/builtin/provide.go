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

func Provide(cfg *istructsmem.AppConfigType, buildInfo *debug.BuildInfo, asp istorage.IAppStorageProvider) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(istructs.QNameCommandCUD, istructsmem.NullCommandExec))

	// Deprecated: use c.sys.CUD instead. Kept for backward compatibility only
	// to import via ImportBO
	cfg.Resources.Add(istructsmem.NewCommandFunction(QNameCommandInit, istructsmem.NullCommandExec))

	provideRefIntegrityValidation(cfg)
	provideQryModules(cfg, buildInfo)

	provideQryEcho(cfg)
	provideQryGRCount(cfg)
	proivideRenameQName(cfg, asp)
	provideSysIsActiveValidation(cfg)
}
