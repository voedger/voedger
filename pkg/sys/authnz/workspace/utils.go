/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package workspace

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func validateWSKindInitializationData(as istructs.IAppStructs, data map[string]interface{}, def appdef.IDef) (err error) {
	reb := as.Events().GetNewRawEventBuilder(
		istructs.NewRawEventBuilderParams{
			GenericRawEventBuilderParams: istructs.GenericRawEventBuilderParams{
				QName: def.QName(),
			},
		},
	)
	aob := reb.ArgumentObjectBuilder()
	aob.PutQName(appdef.SystemField_QName, def.QName())
	aob.PutRecordID(appdef.SystemField_ID, 1)
	if err = coreutils.Marshal(aob, data); err != nil {
		return err
	}
	_, err = aob.Build()
	return
}
