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

func provideQryModules(sr istructsmem.IStatelessResources, buildInfo *debug.BuildInfo) {
	sr.AddQueries(appdef.SysPackagePath, istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "Modules"),
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
			fmt.Fprintf(sb, "path: %s version: %s\n", mod.Path, mod.Version)
		}
		return callback(&qryModulesRR{modules: sb.String()})
	}
}
