/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package engines

import (
	"context"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

func provideStatelessFuncs(resources istructsmem.StatelessResources) iextengine.BuiltInAppExtFuncs {
	funcs := iextengine.BuiltInAppExtFuncs{}
	resources.Resources.Resources(func(qName appdef.QName) {
		res := resources.QueryResource(qName)
		var fn iextengine.BuiltInExtFunc
		switch ifunc := res.(type) {
		case istructs.ICommandFunction:
			fn = func(_ context.Context, io iextengine.IExtensionIO) error {
				execArgs := istructs.ExecCommandArgs{
					CommandPrepareArgs: io.CommandPrepareArgs(),
					State:              io,
					Intents:            io,
				}
				return ifunc.Exec(execArgs)
			}
		case istructs.IQueryFunction:
			fn = func(ctx context.Context, io iextengine.IExtensionIO) error {
				return ifunc.Exec(
					ctx,
					istructs.ExecQueryArgs{
						PrepareArgs: io.QueryPrepareArgs(),
						State:       io,
						Intents:     io,
					},
					io.QueryCallback(),
				)
			}
		default:
			// notest
			panic(fmt.Sprintf("unsupported resource type %T", ifunc))
		}
		fullQName := appdef.NewFullQName(resources.PkgPath, qName.Entity())
		funcs[fullQName] = fn
	})
	for _, syncProjector := range resources.SyncProjectors {
		fullQName := appdef.NewFullQName(resources.PkgPath, syncProjector.Name.Entity())
		funcs[fullQName] = func(_ context.Context, io iextengine.IExtensionIO) error {
			return syncProjector.Func(io.PLogEvent(), io, io)
		}
	}
	for _, asyncProjector := range resources.AsyncProjectors {
		fullQName := appdef.NewFullQName(resources.PkgPath, asyncProjector.Name.Entity())
		funcs[fullQName] = func(_ context.Context, io iextengine.IExtensionIO) error {
			return asyncProjector.Func(io.PLogEvent(), io, io)
		}
	}
	return funcs
}

// provides all built-in extension functions for specified application config
//
// # Panics:
//   - if any extension implementation not found
//   - if any extension package full path is unknown
func provideAppsBuiltInExtFuncs(cfgs istructsmem.AppConfigsType) iextengine.BuiltInExtFuncs {
	funcs := make(iextengine.BuiltInExtFuncs)

	for app, cfg := range cfgs {
		appFuncs := make(iextengine.BuiltInAppExtFuncs)
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
							execArgs := istructs.ExecCommandArgs{
								CommandPrepareArgs: io.CommandPrepareArgs(),
								State:              io,
								Intents:            io,
							}
							return cmd.Exec(execArgs)
						}
					}
				case appdef.TypeKind_Query:
					if query, ok := cfg.Resources.QueryResource(name).(istructs.IQueryFunction); ok {
						fn = func(ctx context.Context, io iextengine.IExtensionIO) error {
							return query.Exec(
								ctx,
								istructs.ExecQueryArgs{
									PrepareArgs: io.QueryPrepareArgs(),
									State:       io,
									Intents:     io,
								},
								io.QueryCallback(),
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
					panic(fmt.Errorf("application «%v»: package «%s» full path is unknown", app, name.Pkg()))
				}

				appFuncs[extName] = fn
			})

		funcs[app] = appFuncs
	}
	return funcs
}
