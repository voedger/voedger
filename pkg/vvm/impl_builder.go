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

func (ab VVMAppsBuilder) PrepareAppsExtensionPoints() map[istructs.AppQName]extensionpoints.IExtensionPoint {
	seps := map[istructs.AppQName]extensionpoints.IExtensionPoint{}
	for appQName := range ab {
		seps[appQName] = extensionpoints.NewRootExtensionPoint()
	}
	return seps
}

func buillAppFromPackagesFS(fses []parser.PackageFS, adf appdef.IAppDefBuilder) error {
	packageSchemaASTs := []*parser.PackageSchemaAST{}
	for _, fs := range fses {
		packageSchemaAST, err := parser.ParsePackageDir(fs.Path, fs.FS, ".")
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

func (ab VVMAppsBuilder) BuiltInAppsPackages(cfgs istructsmem.AppConfigsType, apis apps.APIs, appsEPs map[istructs.AppQName]extensionpoints.IExtensionPoint) (builtInAppsPackages []BuiltInAppsPackages, err error) {
	for appQName, appBuilder := range ab {
		adb := appdef.New()
		appEPs := appsEPs[appQName]
		cfg := cfgs.AddConfig(appQName, adb)
		builtInAppDef := appBuilder(apis, cfg, adb, appEPs)
		if err := buillAppFromPackagesFS(builtInAppDef.Packages, adb); err != nil {
			return nil, err
		}
		biltInAppPackages := BuiltInAppsPackages{
			BuiltInApp: apppartsctl.BuiltInApp{
				Name:           appQName,
				PartsCount:     builtInAppDef.PartsCount,
				EnginePoolSize: builtInAppDef.EnginePoolSize,
			},
			Packages: builtInAppDef.Packages,
		}
		if biltInAppPackages.Def, err = adb.Build(); err != nil {
			return nil, err
		}
		builtInAppsPackages = append(builtInAppsPackages, biltInAppPackages)
	}
	return builtInAppsPackages, nil
}
