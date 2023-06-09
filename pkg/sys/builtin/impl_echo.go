/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package builtin

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
)

func (e *echoRR) AsString(string) string { return e.text }

func ProvideQryEcho(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "Echo"),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "EchoParams")).
			AddField("Text", appdef.DataKind_string, true).(appdef.IDef).QName(),
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "EchoResult")).
			AddField("Res", appdef.DataKind_string, true).(appdef.IDef).QName(),
		func(_ context.Context, _ istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
			text := args.ArgumentObject.AsString("Text")
			return callback(&echoRR{text: text})
		},
	))
}
