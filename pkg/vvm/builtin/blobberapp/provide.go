/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package blobberapp

import (
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/sys/sysprovide"
	builtinapps "github.com/voedger/voedger/pkg/vvm/builtin"
)

func Provide() builtinapps.Builder {
	return func(apis builtinapps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) builtinapps.Def {
		sysPackageFS := sysprovide.Provide(cfg) // need to generate AppWorkspaces only
		blobberAppPackageFS := parser.PackageFS{
			Path: BlobberAppFQN,
			FS:   blobberSchemaFS,
		}
		return builtinapps.Def{
			AppQName: istructs.AppQName_sys_blobber,
			Packages: []parser.PackageFS{sysPackageFS, blobberAppPackageFS},
			AppDeploymentDescriptor: appparts.AppDeploymentDescriptor{
				NumParts:       DefDeploymentPartsCount,
				EnginePoolSize: DefDeploymentEnginePoolSize,
			},
		}
	}
}
