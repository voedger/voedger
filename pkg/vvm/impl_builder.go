/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
)

func (hap VVMAppsBuilder) Add(appQName istructs.AppQName, builder apps.AppBuilder) {
	builders := hap[appQName]
	builders = append(builders, builder)
	hap[appQName] = builders
}

func (hap VVMAppsBuilder) PrepareAppsExtensionPoints() map[istructs.AppQName]extensionpoints.IExtensionPoint {
	seps := map[istructs.AppQName]extensionpoints.IExtensionPoint{}
	for appQName := range hap {
		seps[appQName] = extensionpoints.NewRootExtensionPoint()
	}
	return seps
}

func buildSchemasASTs(adf appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) {
	packageSchemaASTs, err := ReadPackageSchemaAST(ep)
	if err != nil {
		panic(err)
	}
	appSchemaAST, err := parser.BuildAppSchema(packageSchemaASTs)
	if err != nil {
		panic(err)
	}
	if err := parser.BuildAppDefs(appSchemaAST, adf); err != nil {
		panic(err)
	}
}

func (hap VVMAppsBuilder) Build(cfgs istructsmem.AppConfigsType, apis apps.APIs, appsEPs map[istructs.AppQName]extensionpoints.IExtensionPoint) (vvmApps VVMApps) {
	for appQName, appBuilders := range hap {
		adf := appdef.New()
		appEPs := appsEPs[appQName]
		cfg := cfgs.AddConfig(appQName, adf)
		for _, builder := range appBuilders {
			builder(apis, cfg, adf, appEPs)
		}
		buildSchemasASTs(adf, appEPs)
		vvmApps = append(vvmApps, appQName)
		appDef, err := adf.Build()
		if err != nil {
			panic(err)
		}
		appDef.Types(func(t appdef.IType) {
			switch t.Kind() {
			case appdef.TypeKind_Command:
				cmd := t.(appdef.ICommand)
				cmdResource := cfg.Resources.QueryResource(cmd.QName()).(istructs.ICommandFunction)
				resQName := appdef.NullQName
				if cmd.Result() != nil {
					resQName = cmd.Result().QName()
				}
				paramQName := appdef.NullQName
				if cmd.Param() != nil {
					paramQName = cmd.Param().QName()
				}
				unloggedParamQName := appdef.NullQName
				if cmd.UnloggedParam() != nil {
					unloggedParamQName = cmd.UnloggedParam().QName()
				}
				istructsmem.ReplaceCommandDefinitions(cmdResource, paramQName, unloggedParamQName, resQName)
			case appdef.TypeKind_Query:
				if t.QName() == qNameQueryCollection {
					return
				}
				query := t.(appdef.IQuery)
				queryResource := cfg.Resources.QueryResource(query.QName()).(istructs.IQueryFunction)
				paramQName := appdef.NullQName
				if query.Param() != nil {
					paramQName = query.Param().QName()
				}
				istructsmem.ReplaceQueryDefinitions(queryResource, paramQName, query.Result().QName())
			}
		})
	}
	return vvmApps
}
