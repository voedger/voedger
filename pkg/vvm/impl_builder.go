/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	istructsmem "github.com/voedger/voedger/pkg/istructsmem"
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
	epPackageSchemasASTs := ep.ExtensionPoint(apps.EPPackageSchemasASTs)
	packageSchemaASTs := []*parser.PackageSchemaAST{}
	epPackageSchemasASTs.Iterate(func(eKey extensionpoints.EKey, value interface{}) {
		qualifiedPackageName := eKey.(string)
		packageFilesSchemasASTsEP := value.(extensionpoints.IExtensionPoint)
		packageFilesSchemasASTs := []*parser.FileSchemaAST{}
		packageFilesSchemasASTsEP.Iterate(func(eKey extensionpoints.EKey, value interface{}) {
			fileSchemaAST := value.(*parser.FileSchemaAST)
			packageFilesSchemasASTs = append(packageFilesSchemasASTs, fileSchemaAST)
		})
		packageSchemaAST, err := parser.MergeFileSchemaASTs(qualifiedPackageName, packageFilesSchemasASTs)
		if err != nil {
			panic(err)
		}
		packageSchemaASTs = append(packageSchemaASTs, packageSchemaAST)
	})
	packageSchemas, err := parser.MergePackageSchemas(packageSchemaASTs)
	if err != nil {
		panic(err)
	}
	if err := parser.BuildAppDefs(packageSchemas, adf); err != nil {
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
				istructsmem.ReplaceCommandDefinitions(cmdResource, cmd.Arg().QName(), cmd.UnloggedArg().QName(), cmd.Result().QName())
			case appdef.TypeKind_Query:
				if t.QName() == qNameQueryCollection {
					return
				}
				query := t.(appdef.IQuery)
				queryResource := cfg.Resources.QueryResource(query.QName()).(istructs.IQueryFunction)
				istructsmem.ReplaceQueryDefinitions(queryResource, query.Arg().QName(), query.Result().QName())
			}
		})
	}
	return vvmApps
}
