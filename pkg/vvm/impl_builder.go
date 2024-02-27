/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apppartsctl"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
)

func (ab VVMAppsBuilder) Add(appQName istructs.AppQName, builder apps.AppBuilder) {
	if _, ok := ab[appQName]; ok {
		panic(appQName.String() + " builder already added")
	}
	ab[appQName] = builder
}

func (hap VVMAppsBuilder) PrepareAppsExtensionPoints() map[istructs.AppQName]extensionpoints.IExtensionPoint {
	seps := map[istructs.AppQName]extensionpoints.IExtensionPoint{}
	for appQName := range hap {
		seps[appQName] = extensionpoints.NewRootExtensionPoint()
	}
	return seps
}

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

func (hap VVMAppsBuilder) Build(cfgs istructsmem.AppConfigsType, apis apps.APIs, appsEPs map[istructs.AppQName]extensionpoints.IExtensionPoint) (builtInApps []apppartsctl.BuiltInApp, err error) {
	for appQName, appBuilder := range hap {
		adb := appdef.New()
		appEPs := appsEPs[appQName]
		cfg := cfgs.AddConfig(appQName, adb)
		builtInAppDef := appBuilder(apis, cfg, adb, appEPs)
		if err := buillAppFromPackagesFS(builtInAppDef.Packages, adb); err != nil {
			return nil, err
		}
		builtInApp := apppartsctl.BuiltInApp{
			Name:           appQName,
			PartsCount:     builtInAppDef.PartsCount,
			EnginePoolSize: builtInAppDef.EnginePoolSize,
		}
		if builtInApp.Def, err = adb.Build(); err != nil {
			return nil, err
		}
		builtInApps = append(builtInApps, builtInApp)
	}
	return builtInApps, nil
}
