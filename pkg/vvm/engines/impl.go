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

	for path, cmd := range statelessResources.Commands {
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
	}

	for path, qry := range statelessResources.Queries {
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
	}

	for path, projector := range statelessResources.Projectors {
		fullQName := appdef.NewFullQName(path, projector.Name.Entity())
		funcs[fullQName] = func(_ context.Context, io iextengine.IExtensionIO) error {
			return projector.Func(io.PLogEvent(), io, io)
		}
	}

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
		writeFuncs(cfg, appFuncs)
		writeProjectors(cfg, appFuncs)
		writeJobs(cfg, appFuncs)

		// sync projectors
		for _, syncProjector := range cfg.SyncProjectors() {
			sp := syncProjector
			fn := func(_ context.Context, io iextengine.IExtensionIO) error {
				return sp.Func(io.PLogEvent(), io, io)
			}
			extName := extName(syncProjector.Name, cfg)
			appFuncs[extName] = fn
		}

		// async projectors
		for _, asyncProjector := range cfg.AsyncProjectors() {
			asp := asyncProjector
			fn := func(_ context.Context, io iextengine.IExtensionIO) error {
				return asp.Func(io.PLogEvent(), io, io)
			}
			extName := extName(asyncProjector.Name, cfg)
			appFuncs[extName] = fn
		}

		funcs[app] = appFuncs
	}
	return funcs
}

func writeJobs(cfg *istructsmem.AppConfigType, appFuncs iextengine.BuiltInExtFuncs) {
	for _, builtinJob := range cfg.BuiltingJobs() {
		fn := func(_ context.Context, io iextengine.IExtensionIO) error {
			bj := builtinJob
			return bj.Func(io, io)
		}
		extName := extName(builtinJob.Name, cfg)
		appFuncs[extName] = fn
	}
}

func writeProjectors(cfg *istructsmem.AppConfigType, appFuncs iextengine.BuiltInExtFuncs) {
	write := func(projectors istructs.Projectors) {
		for _, projector := range projectors {
			p := projector
			fn := func(_ context.Context, io iextengine.IExtensionIO) error {
				return p.Func(io.PLogEvent(), io, io)
			}
			extName := extName(p.Name, cfg)
			appFuncs[extName] = fn
		}
	}
	write(cfg.SyncProjectors())
	write(cfg.AsyncProjectors())
}

func writeFuncs(cfg *istructsmem.AppConfigType, appFuncs iextengine.BuiltInExtFuncs) {
	for qName := range cfg.Resources.Resources {
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
	}
}
