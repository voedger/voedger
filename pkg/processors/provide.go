/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package processors

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func ProvideJSONFuncParamsDef(appDef appdef.IAppDefBuilder) {
	appDef.AddObject(istructs.QNameJSON).AddField(Field_JSONDef_Body, appdef.DataKind_string, true, appdef.MaxLen(fieldBodyLen))
}
