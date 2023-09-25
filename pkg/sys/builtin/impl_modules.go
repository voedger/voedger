/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package builtin

import (
	"bytes"
	"context"
	"fmt"
	"runtime/debug"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
)

func provideQryModules(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, buildInfo *debug.BuildInfo) {
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "Modules"),
		appdef.NullQName,
		appDefBuilder.AddObject(appdef.NewQName(appdef.SysPackage, "ModulesResult")).
			AddField("Modules", appdef.DataKind_string, true).(appdef.IDef).QName(),
		provideQryModulesExec(buildInfo),
	))
}

type qryModulesRR struct {
	istructs.NullObject
	modules string
}

func provideQryModulesExec(buildInfo *debug.BuildInfo) istructsmem.ExecQueryClosure {
	return func(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		sb := bytes.NewBufferString("")
		for _, mod := range buildInfo.Deps {
			sb.WriteString(fmt.Sprintf("path: %s version: %s\n", mod.Path, mod.Version))
		}
		return callback(&qryModulesRR{modules: sb.String()})
	}
}
