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

func buildAppFromPackagesFS(fses []parser.PackageFS, adf appdef.IAppDefBuilder) error {
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

func (ab VVMAppsBuilder) BuiltInAppsPackages(cfgs istructsmem.AppConfigsType, apis apps.APIs, appsEPs map[istructs.AppQName]extensionpoints.IExtensionPoint) (builtInAppsPackages []BuiltInAppPackages, err error) {
	for appQName, appBuilder := range ab {
		adb := appdef.New()
		appEPs := appsEPs[appQName]
		cfg := cfgs.AddConfig(appQName, adb)
		builtInAppDef := appBuilder(apis, cfg, appEPs)
		if err := buildAppFromPackagesFS(builtInAppDef.Packages, adb); err != nil {
			return nil, err
		}
		// query IAppStructs to build IAppDef only once - on AppConfigType.preapre()
		_, err = apis.IAppStructsProvider.AppStructs(appQName)
		if err != nil {
			return nil, err
		}
		builtInAppPackages := BuiltInAppPackages{
			BuiltInApp: apppartsctl.BuiltInApp{
				Name:           appQName,
				NumParts:       builtInAppDef.NumParts,
				EnginePoolSize: builtInAppDef.EnginePoolSize,
				Def:            cfg.AppDef,
			},
			Packages: builtInAppDef.Packages,
		}
		builtInAppsPackages = append(builtInAppsPackages, builtInAppPackages)
	}
	return builtInAppsPackages, nil
}
