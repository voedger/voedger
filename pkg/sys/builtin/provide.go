/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package builtin

import (
	"context"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// для редактирования BO с Web
func ProvideCmdCUD(cfg *istructsmem.AppConfigType) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(istructs.QNameCommandCUD, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))
}

// для импорта через ImportBO
func ProvideCmdInit(cfg *istructsmem.AppConfigType) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(QNameCommandInit, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))
}

// вместо Sync
func ProivdeCmdImport(cfg *istructsmem.AppConfigType) {
	cfg.Resources.Add(istructsmem.NewCommandFunction(QNameCommandImport, appdef.NullQName, appdef.NullQName, appdef.NullQName, istructsmem.NullCommandExec))
}

func ProvideRefIntegrityValidator() istructs.CUDValidator {
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
