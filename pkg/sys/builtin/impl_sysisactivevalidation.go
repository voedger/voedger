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
		Match: func(cud istructs.ICUDRow, _ istructs.WSID, _ appdef.QName) bool {
			return cud.IsNew()
		},
		Validate: func(_ context.Context, _ istructs.IAppStructs, cudRow istructs.ICUDRow, _ istructs.WSID, _ appdef.QName, _ istructs.IStateValue) error {
			if !cudRow.AsBool(appdef.SystemField_IsActive) {
				return errors.New("inserting a deactivated record is not allowed")
			}
			return nil
		},
	})
}
