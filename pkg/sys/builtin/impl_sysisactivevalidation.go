/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package builtin

import (
	"context"
	"errors"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func ProvideSysIsActiveValidation(cfg *istructsmem.AppConfigType) {
	cfg.AddCUDValidators(istructs.CUDValidator{
		Match: func(cud istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) bool {
			return cud.IsNew()
		},
		Validate: func(ctx context.Context, appStructs istructs.IAppStructs, cudRow istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) error {
			if !cudRow.AsBool(appdef.SystemField_IsActive) {
				return errors.New("inserting a deactivated record is not allowed")
			}
			return nil
		},
	})
}
