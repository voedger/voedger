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

func provideStatelessFuncs(statelessResources istructsmem.IStatelessResources) iextengine.BuiltInExtFuncs {
	funcs := iextengine.BuiltInExtFuncs{}

	statelessResources.Commands(func(path string, cmd istructs.ICommandFunction) {
		fn := func(_ context.Context, io iextengine.IExtensionIO) error {
			execArgs := istructs.ExecCommandArgs{
				CommandPrepareArgs: io.CommandPrepareArgs(),
				State:              io,
				Intents:            io,
			}
			return cmd.Exec(execArgs)
		}
		fullQName := appdef.NewFullQName(path, cmd.QName().Entity())
		funcs[fullQName] = fn
	})

	statelessResources.Queries(func(path string, qry istructs.IQueryFunction) {
		fn := func(ctx context.Context, io iextengine.IExtensionIO) error {
			return qry.Exec(
				ctx,
				istructs.ExecQueryArgs{
					PrepareArgs: io.QueryPrepareArgs(),
					State:       io,
					Intents:     io,
				},
				io.QueryCallback(),
			)
		}
		fullQName := appdef.NewFullQName(path, qry.QName().Entity())
		funcs[fullQName] = fn
	})

	statelessResources.Projectors(func(path string, projector istructs.Projector) {
		fullQName := appdef.NewFullQName(path, projector.Name.Entity())
		funcs[fullQName] = func(_ context.Context, io iextengine.IExtensionIO) error {
			return projector.Func(io.PLogEvent(), io, io)
		}
	})

	return funcs
}

func extName(qName appdef.QName, cfg *istructsmem.AppConfigType) appdef.FullQName {
	extName := cfg.AppDef.FullQName(qName)
	if extName == appdef.NullFullQName {
		panic(fmt.Errorf("application «%v»: package «%s» full path is unknown", cfg.Name, qName.Pkg()))
	}
	return extName
}

// provides all built-in extension functions for specified application config
//
// # Panics:
//   - if any extension implementation not found
//   - if any extension package full path is unknown
func provideAppsBuiltInExtFuncs(cfgs istructsmem.AppConfigsType) iextengine.BuiltInAppExtFuncs {
	funcs := make(iextengine.BuiltInAppExtFuncs)

	for app, cfg := range cfgs {
		appFuncs := make(iextengine.BuiltInExtFuncs)
		cfg.Resources.Resources(func(qName appdef.QName) {
			ires := cfg.Resources.QueryResource(qName)
			var fn iextengine.BuiltInExtFunc
			switch resource := ires.(type) {
			case istructs.ICommandFunction:
				fn = func(_ context.Context, io iextengine.IExtensionIO) error {
					execArgs := istructs.ExecCommandArgs{
						CommandPrepareArgs: io.CommandPrepareArgs(),
						State:              io,
						Intents:            io,
					}
					return resource.Exec(execArgs)
				}
			case istructs.IQueryFunction:
				fn = func(ctx context.Context, io iextengine.IExtensionIO) error {
					return resource.Exec(
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
			extName := extName(qName, cfg)
			appFuncs[extName] = fn
		})

		for _, syncProjector := range cfg.SyncProjectors() {
			sp := syncProjector
			fn := func(_ context.Context, io iextengine.IExtensionIO) error {
				return sp.Func(io.PLogEvent(), io, io)
			}
			extName := extName(syncProjector.Name, cfg)
			appFuncs[extName] = fn
		}
		for _, asyncProjector := range cfg.AsyncProjectors() {
			sp := asyncProjector
			fn := func(_ context.Context, io iextengine.IExtensionIO) error {
				return sp.Func(io.PLogEvent(), io, io)
			}
			extName := extName(asyncProjector.Name, cfg)
			appFuncs[extName] = fn
		}

		funcs[app] = appFuncs
	}
	return funcs
}
