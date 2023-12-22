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

// func buildSchemasASTs(adf appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) error {
// 	packageSchemaASTs, err := ReadPackageSchemaAST(ep)
// 	if err != nil {
// 		return err
// 	}
// 	appSchemaAST, err := parser.BuildAppSchema(packageSchemaASTs)
// 	if err != nil {
// 		return err
// 	}
// 	return parser.BuildAppDefs(appSchemaAST, adf)
// }

func buillAppFromPackagesFS(fses []parser.PackageFS, adf appdef.IAppDefBuilder) error {
	packageSchemaASTs := []*parser.PackageSchemaAST{}
	for _, fs := range fses {
		packageSchemaAST, err := parser.ParsePackageDir(fs.QualifiedPackageName, fs.FS, ".")
		if err != nil {
			return err
		}
		packageSchemaASTs = append(packageSchemaASTs, packageSchemaAST)
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
		appPackagesFSes := []parser.PackageFS{}
		for _, appBuilder := range appBuilders {
			appPackagesFSes = append(appPackagesFSes, appBuilder(apis, cfg, adf, appEPs)...)
		}
		if err := buillAppFromPackagesFS(appPackagesFSes, adf); err != nil {
			return nil, err
		}
		// if err := buildSchemasASTs(adf, appEPs); err != nil {
		// 	return nil, err
		// }
		vvmApps = append(vvmApps, appQName)
		if _, err := adf.Build(); err != nil {
			return nil, err
		}
	}
	return vvmApps, nil
}
