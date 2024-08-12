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
	"github.com/voedger/voedger/pkg/sys/sysprovide"
)

func Provide() apps.AppBuilder {
	return func(apis apps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) apps.BuiltInAppDef {
		sysPackageFS := sysprovide.Provide(cfg)
		routerAppPackageFS := parser.PackageFS{
			Path: RouterAppFQN,
			FS:   routerAppSchemaFS,
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
