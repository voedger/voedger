/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vvm

import (
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
	builtinapps "github.com/voedger/voedger/pkg/vvm/builtin"
)

// appQName->*parser.AppSchemaAST
// used in tests only
var testAppSchemaCache sync.Map = sync.Map{}

func (ab VVMAppsBuilder) Add(appQName appdef.AppQName, builder builtinapps.Builder) {
	if _, ok := ab[appQName]; ok {
		panic(appQName.String() + " builder already added")
	}
	ab[appQName] = builder
}

func buildAppFromPackagesFS(appQName appdef.AppQName, fses []parser.PackageFS, adf appdef.IAppDefBuilder, cacheSchemas bool) (err error) {
	var appSchemaAST *parser.AppSchemaAST
	if cacheSchemas {
		appSchemaASTIntf, ok := testAppSchemaCache.Load(appQName)
		if ok {
			appSchemaAST = appSchemaASTIntf.(*parser.AppSchemaAST)
		}
	}
	if appSchemaAST == nil {
		packageSchemaASTs := []*parser.PackageSchemaAST{}
		for _, fs := range fses {
			packageSchemaAST, err := parser.ParsePackageDir(fs.Path, fs.FS, ".")
			if err != nil {
				return err
			}
			packageSchemaASTs = append(packageSchemaASTs, packageSchemaAST)
		}
		appSchemaAST, err = parser.BuildAppSchema(packageSchemaASTs)
		if err != nil {
			return err
		}
		if cacheSchemas {
			testAppSchemaCache.Store(appQName, appSchemaAST)
		}
	}
	return parser.BuildAppDefs(appSchemaAST, adf)
}

func (ab VVMAppsBuilder) BuildAppsArtefacts(apis builtinapps.APIs, emptyCfgs AppConfigsTypeEmpty,
	appsEPs map[appdef.AppQName]extensionpoints.IExtensionPoint) (builtinAppsArtefacts BuiltInAppsArtefacts, err error) {
	builtinAppsArtefacts.AppConfigsType = istructsmem.AppConfigsType(emptyCfgs)
	for appQName, appBuilder := range ab {
		appEPs := appsEPs[appQName]
		adb := builder.New()
		cfg := builtinAppsArtefacts.AppConfigsType.AddBuiltInAppConfig(appQName, adb)
		builtInAppDef := appBuilder(apis, cfg, appEPs)
		cfg.SetNumAppWorkspaces(builtInAppDef.NumAppWorkspaces)
		cacheSchemas := builtInAppDef.CacheAppSchemASTInTests && coreutils.IsTest()
		if err := buildAppFromPackagesFS(appQName, builtInAppDef.Packages, adb, cacheSchemas); err != nil {
			return builtinAppsArtefacts, err
		}

		// query IAppStructs to build IAppDef only once - on AppConfigType.prepare()
		// that need to catch errors from IAppDefBuilder etc here, not after VVM launch
		// also we need ready-to-use IAppStorage to read a dictionary QName->QNameID
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
