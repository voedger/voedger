/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package blobberapp

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
		sysPackageFS := sysprovide.Provide(cfg) // need to generate AppWorkspaces only
		blobberAppPackageFS := parser.PackageFS{
			Path: BlobberAppFQN,
			FS:   blobberSchemaFS,
		}
		return apps.BuiltInAppDef{
			AppQName: istructs.AppQName_sys_blobber,
			Packages: []parser.PackageFS{sysPackageFS, blobberAppPackageFS},
			AppDeploymentDescriptor: appparts.AppDeploymentDescriptor{
				NumParts:       DefDeploymentPartsCount,
				EnginePoolSize: DefDeploymentEnginePoolSize,
			},
		}
	}
}
