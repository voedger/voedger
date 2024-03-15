/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package vvm

import (
	"context"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

// provides all built-in extension functions for all specified applications
//
// # Panics:
//   - if any extension implementation not found
func provideAppsBuiltInExtFuncs(cfgs istructsmem.AppConfigsType) iextengine.BuiltInExtFuncs {
	funcs := make(iextengine.BuiltInExtFuncs)
	for app := range cfgs {
		cfg := cfgs.GetConfig(app)

		cfg.AppDef.Extensions(
			func(ext appdef.IExtension) {
				if ext.Engine() != appdef.ExtensionEngineKind_BuiltIn {
					return
				}
				name := ext.QName()

				var fn iextengine.BuiltInExtFunc

				switch ext.Kind() {
				case appdef.TypeKind_Command:
					if cmd, ok := cfg.Resources.QueryResource(name).(istructs.ICommandFunction); ok {
						fn = func(_ context.Context, io iextengine.IExtensionIO) error {
							return cmd.Exec(istructs.ExecCommandArgs{State: io, Intents: io})
						}
					}
				case appdef.TypeKind_Query:
					if query, ok := cfg.Resources.QueryResource(name).(istructs.IQueryFunction); ok {
						fn = func(ctx context.Context, io iextengine.IExtensionIO) error {
							return query.Exec(
								ctx,
								istructs.ExecQueryArgs{State: io},
								// TODO: add query result handler
								func(istructs.IObject) error { return nil },
							)
						}
					}
				case appdef.TypeKind_Projector:
					var prj istructs.Projector
					if ext.(appdef.IProjector).Sync() {
						prj = cfg.SyncProjectors()[name]
					} else {
						prj = cfg.AsyncProjectors()[name]
					}
					if prj.Name != appdef.NullQName {
						fn = func(_ context.Context, io iextengine.IExtensionIO) error {
							return prj.Func(io.PLogEvent(), io, io)
						}
					}
				}

				if fn == nil {
					panic(fmt.Errorf("application «%v»: %v implementation not found", app, ext))
				}

				extName := cfg.AppDef.FullQName(name)
				if extName == appdef.NullFullQName {
					panic(fmt.Errorf("application «%v»: package %v full path is unknown", app, name.Pkg()))
				}

				funcs[extName] = fn
			})
	}
	return funcs
}
