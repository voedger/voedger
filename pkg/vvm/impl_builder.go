/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
)

func (ab VVMAppsBuilder) Add(appQName appdef.AppQName, builder apps.AppBuilder) {
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

func (ab VVMAppsBuilder) BuildAppsArtefacts(apis apps.APIs, emptyCfgs AppConfigsTypeEmpty,
	appsEPs map[appdef.AppQName]extensionpoints.IExtensionPoint) (builtinAppsArtefacts BuiltInAppsArtefacts, err error) {
	builtinAppsArtefacts.AppConfigsType = istructsmem.AppConfigsType(emptyCfgs)
	for appQName, appBuilder := range ab {
		appEPs := appsEPs[appQName]
		adb := appdef.New()
		cfg := builtinAppsArtefacts.AppConfigsType.AddBuiltInAppConfig(appQName, adb)
		builtInAppDef := appBuilder(apis, cfg, appEPs)
		cfg.SetNumAppWorkspaces(builtInAppDef.NumAppWorkspaces)
		if err := buildAppFromPackagesFS(builtInAppDef.Packages, adb); err != nil {
			return builtinAppsArtefacts, err
		}
		// query IAppStructs to build IAppDef only once - on AppConfigType.prepare()
		// это надо чтобы отловить ошибки IAppDefBuilder и проч
		// также там нуже уже готовый IAppStorage чтобы вычитать QName->QNameID
		if _, err = apis.IAppStructsProvider.BuiltIn(appQName); err != nil {
			return builtinAppsArtefacts, err
		}
		builtInAppPackages := BuiltInAppPackages{
			BuiltInApp: appparts.BuiltInApp{
				Name:                    appQName,
				Def:                     cfg.AppDef,
				AppDeploymentDescriptor: builtInAppDef.AppDeploymentDescriptor,
			},
			Packages: builtInAppDef.Packages,
		}
		builtinAppsArtefacts.builtInAppPackages = append(builtinAppsArtefacts.builtInAppPackages, builtInAppPackages)
	}
	return builtinAppsArtefacts, nil
}
