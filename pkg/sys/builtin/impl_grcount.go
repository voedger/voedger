/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package builtin

import (
	"context"
	"runtime"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
)

type grcountRR struct {
	istructs.NullObject
}

func (e *grcountRR) AsInt32(string) int32 { return int32(runtime.NumGoroutine()) }

func ProvideQryGRCount(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder) {
	cfg.Resources.Add(istructsmem.NewQueryFunction(
		appdef.NewQName(appdef.SysPackage, "GRCount"),
		appdef.NullQName,
		appDefBuilder.AddStruct(appdef.NewQName(appdef.SysPackage, "GRCountResult"), appdef.DefKind_Object).
			AddField("NumGoroutines", appdef.DataKind_int32, true).QName(),
		func(_ context.Context, _ istructs.IQueryFunction, _ istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
			return callback(&grcountRR{})
		},
	))
}
