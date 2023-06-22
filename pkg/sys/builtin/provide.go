/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package builtin

import (
	"context"
	"net/http"
	"runtime/debug"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfg *istructsmem.AppConfigType, adf appdef.IAppDefBuilder, buildInfo *debug.BuildInfo) {
	// to edit BO fron Web
	cfg.Resources.Add(istructsmem.NewCommandFunction(istructs.QNameCommandCUD, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))

	// to import via ImportBO
	cfg.Resources.Add(istructsmem.NewCommandFunction(QNameCommandInit, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))

	// instead of sync
	cfg.Resources.Add(istructsmem.NewCommandFunction(QNameCommandImport, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))

	cfg.AddCUDValidators(provideRefIntegrityValidator())
	provideQryModules(cfg, adf, buildInfo)

}

func provideRefIntegrityValidator() istructs.CUDValidator {
	return istructs.CUDValidator{
		MatchFunc: func(qName appdef.QName) bool {
			return true
		},
		Validate: func(ctx context.Context, appStructs istructs.IAppStructs, cudRow istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) (err error) {
			if coreutils.IsDummyWS(wsid) || cmdQName == QNameCommandImport || cmdQName == QNameCommandInit {
				return nil
			}
			return coreutils.WrapSysError(istructsmem.CheckRefIntegrity(cudRow, appStructs, wsid), http.StatusBadRequest)
		},
	}
}
