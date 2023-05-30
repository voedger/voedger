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
	pars := appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "EchoParams"))
	pars.AddField("Text", appdef.DataKind_string, true)
	res := appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "EchoResult"))
	res.AddField("Res", appdef.DataKind_string, true)
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "Echo"), pars.QName(), res.QName(),
		func(_ context.Context, _ istructs.IQueryFunction, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
			text := args.ArgumentObject.AsString("Text")
			return callback(&echoRR{text: text})
		},
	))
}
