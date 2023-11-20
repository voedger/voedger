/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package processors

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func ProvideRawObject(appDef appdef.IAppDefBuilder) {
	appDef.AddObject(istructs.QNameRaw).AddField(Field_RawDef_Body, appdef.DataKind_raw, true)
}
