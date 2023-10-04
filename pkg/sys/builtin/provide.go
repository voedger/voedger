/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package builtin

import (
	"context"
	"net/http"
	"runtime/debug"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfg *istructsmem.AppConfigType, adf appdef.IAppDefBuilder, buildInfo *debug.BuildInfo, asp istorage.IAppStorageProvider,
	ep extensionpoints.IExtensionPoint) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(istructs.QNameCommandCUD, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))

	// Deprecated: use c.sys.CUD instead. Kept for backward compatibility only
	// to import via ImportBO
	cfg.Resources.Add(istructsmem.NewCommandFunction(QNameCommandInit, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))

	cfg.AddCUDValidators(provideRefIntegrityValidator())
	provideQryModules(cfg, adf, buildInfo)

	provideQryEcho(cfg, adf, ep)
	provideQryGRCount(cfg, adf)
	proivideRenameQName(cfg, adf, asp)
}

func provideRefIntegrityValidator() istructs.CUDValidator {
	return istructs.CUDValidator{
		MatchFunc: func(qName appdef.QName) bool {
			return true
		},
		Validate: func(ctx context.Context, appStructs istructs.IAppStructs, cudRow istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) (err error) {
			if coreutils.IsDummyWS(wsid) || cmdQName == QNameCommandInit {
				return nil
			}
			return coreutils.WrapSysError(istructsmem.CheckRefIntegrity(cudRow, appStructs, wsid), http.StatusBadRequest)
		},
	}
}
