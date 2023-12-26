/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package routerapp

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/smtp"
)

func Provide(smtpCfg smtp.Cfg) apps.AppBuilder {
	return func(apis apps.APIs, cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, ep extensionpoints.IExtensionPoint) apps.AppPackages {
		sysPackageFS := sys.Provide(cfg, appDefBuilder, smtpCfg, ep, nil, apis.TimeFunc, apis.ITokens, apis.IFederation, apis.IAppStructsProvider, apis.IAppTokensFactory,
			apis.NumCommandProcessors, nil, apis.IAppStorageProvider)
		routerAppPackageFS := parser.PackageFS{
			QualifiedPackageName: RouterAppFQN,
			FS:                   routerAppSchemaFS,
		}
		return apps.AppPackages{
			AppQName: istructs.AppQName_sys_router,
			Packages: []parser.PackageFS{sysPackageFS, routerAppPackageFS},
		}
	}
}

// Returns router application definition
func AppDef() appdef.IAppDef {
	appDef, err := apps.BuildAppDefFromFS(RouterAppFQN, routerAppSchemaFS, ".")
	if err != nil {
		panic(err)
	}
	return appDef
}

// Returns router partitions count
func PartsCount() int { return 1 }

// Returns router engines pool sizes
func EnginePoolSize() [cluster.ProcessorKind_Count]int {
	return [cluster.ProcessorKind_Count]int{1, 1, 1}
}
