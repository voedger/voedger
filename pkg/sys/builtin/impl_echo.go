/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package builtin

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
)

func (e *echoRR) AsString(string) string { return e.text }

func provideQryEcho(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) {
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "Echo"),
		appdef.NullQName,
		appdef.NullQName,
		// appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "EchoParams")).
		// 	AddField("Text", appdef.DataKind_string, true).(appdef.IType).QName(),
		// appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "EchoResult")).
		// 	AddField("Res", appdef.DataKind_string, true).(appdef.IType).QName(),
		func(_ context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
			text := args.ArgumentObject.AsString("Text")
			return callback(&echoRR{text: text})
		},
	))
	apps.Parse(schemaBuiltinFS, appdef.SysPackage, ep)
}
