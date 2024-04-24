/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
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

func (ab VVMAppsBuilder) BuildAppsArtefacts(apis apps.APIs, emptyCfgs AppConfigsTypeEmpty) (appsArtefacts AppsArtefacts, err error) {
	appsArtefacts.appEPs = map[istructs.AppQName]extensionpoints.IExtensionPoint{}
	appsArtefacts.AppConfigsType = istructsmem.AppConfigsType(emptyCfgs)
	for appQName, appBuilder := range ab {
		appEPs := extensionpoints.NewRootExtensionPoint()
		appsArtefacts.appEPs[appQName] = appEPs
		adb := appdef.New()
		cfg := appsArtefacts.AppConfigsType.AddConfig(appQName, adb)
		builtInAppDef := appBuilder(apis, cfg, appEPs)
		cfg.SetNumAppWorkspaces(builtInAppDef.NumAppWorkspaces)
		if err := buildAppFromPackagesFS(builtInAppDef.Packages, adb); err != nil {
			return appsArtefacts, err
		}
		// query IAppStructs to build IAppDef only once - on AppConfigType.prepare()
		_, err = apis.IAppStructsProvider.AppStructs(appQName)
		if err != nil {
			return appsArtefacts, err
		}
		builtInAppPackages := BuiltInAppPackages{
			BuiltInApp: appparts.BuiltInApp{
				Name:                    appQName,
				Def:                     cfg.AppDef,
				AppDeploymentDescriptor: builtInAppDef.AppDeploymentDescriptor,
			},
			Packages: builtInAppDef.Packages,
		}
		appsArtefacts.builtInAppPackages = append(appsArtefacts.builtInAppPackages, builtInAppPackages)
	}
	return appsArtefacts, nil
}
