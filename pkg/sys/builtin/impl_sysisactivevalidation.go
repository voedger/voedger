/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package builtin

import (
	"context"
	"errors"

	"github.com/untillpro/goutils/iterate"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func provideSysIsActiveValidation(cfg *istructsmem.AppConfigType) {
	cfg.AddCUDValidators(denyIsActiveAndOtherFieldsMixing)
}

var denyIsActiveAndOtherFieldsMixing = istructs.CUDValidator{
	Match: func(cud istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) bool {
		return !cud.IsNew()
	},
	Validate: func(ctx context.Context, appStructs istructs.IAppStructs, cudRow istructs.ICUDRow, wsid istructs.WSID, cmdQName appdef.QName) error {
		sysIsActiveUpdating := false
		hasOnlySystemFields := true
		fields := []string{}
		isActiveAndOtherFieldsMixedOnUpdate, _, _ := iterate.FindFirstMap(cudRow.ModifiedFields, func(fieldName string, _ interface{}) bool {
			fields = append(fields, fieldName)
			if !appdef.IsSysField(fieldName) {
				hasOnlySystemFields = false
			} else if fieldName == appdef.SystemField_IsActive {
				sysIsActiveUpdating = true
			}
			if sysIsActiveUpdating && !hasOnlySystemFields {
				return true
			}
			return false
		})
		if isActiveAndOtherFieldsMixedOnUpdate {
			return errors.New("updating other fields is not allowed if sys.IsActive is updating")
		}
		return nil
	},
}

