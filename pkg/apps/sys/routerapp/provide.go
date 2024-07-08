/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package routerapp

import (
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/smtp"
)

func Provide() apps.AppBuilder {
	return func(apis apps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) apps.BuiltInAppDef {
		sysPackageFS := sys.Provide(cfg, smtp.Cfg{}, ep, nil, apis.TimeFunc, apis.ITokens, apis.IFederation, apis.IAppStructsProvider, apis.IAppTokensFactory,
			nil, apis.IAppStorageProvider)
		routerAppPackageFS := parser.PackageFS{
			PackageFQN: RouterAppFQN,
			FS:         routerAppSchemaFS,
		}
		return apps.BuiltInAppDef{
			AppQName: istructs.AppQName_sys_router,
			Packages: []parser.PackageFS{sysPackageFS, routerAppPackageFS},
			AppDeploymentDescriptor: appparts.AppDeploymentDescriptor{
				NumParts:       DefDeploymentPartsCount,
				EnginePoolSize: DefDeploymentEnginePoolSize,
			},
		}
	}
}
