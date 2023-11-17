/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"fmt"

	"github.com/untillpro/goutils/iterate"
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

func buildSchemasASTs(adf appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) error {
	packageSchemaASTs, err := ReadPackageSchemaAST(ep)
	if err != nil {
		return err
	}
	appSchemaAST, err := parser.BuildAppSchema(packageSchemaASTs)
	if err != nil {
		return err
	}
	return parser.BuildAppDefs(appSchemaAST, adf)
}

func (hap VVMAppsBuilder) Build(cfgs istructsmem.AppConfigsType, apis apps.APIs, appsEPs map[istructs.AppQName]extensionpoints.IExtensionPoint) (vvmApps VVMApps, err error) {
	for appQName, appBuilders := range hap {
		adf := appdef.New()
		appEPs := appsEPs[appQName]
		cfg := cfgs.AddConfig(appQName, adf)
		for _, builder := range appBuilders {
			builder(apis, cfg, adf, appEPs)
		}
		if err := buildSchemasASTs(adf, appEPs); err != nil {
			return nil, err
		}
		vvmApps = append(vvmApps, appQName)
		appDef, err := adf.Build()
		if err != nil {
			return nil, err
		}
		err = iterate.ForEachError(appDef.Types, func(t appdef.IType) error {
			if t.Kind() != appdef.TypeKind_Command && t.Kind() != appdef.TypeKind_Query {
				return nil
			}
			resource := cfg.Resources.QueryResource(t.QName())
			if resource.QName() == appdef.NullQName {
				return fmt.Errorf("func %s not found in resources", t.QName())
			}
			switch t.Kind() {
			case appdef.TypeKind_Command:
				cmd := t.(appdef.ICommand)
				cmdResource := resource.(istructs.ICommandFunction)
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
					return nil
				}
				query := t.(appdef.IQuery)
				queryResource := resource.(istructs.IQueryFunction)
				paramQName := appdef.NullQName
				if query.Param() != nil {
					paramQName = query.Param().QName()
				}
				resQName := appdef.NullQName
				if query.Result() != nil {
					resQName = query.Result().QName()
				}
				istructsmem.ReplaceQueryDefinitions(queryResource, paramQName, resQName)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return vvmApps, nil
}
