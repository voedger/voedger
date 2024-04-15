/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package blobberapp

import (
	"github.com/voedger/voedger/pkg/apps"
	"github.com/voedger/voedger/pkg/cluster"
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
			apis.NumCommandProcessors, nil, apis.IAppStorageProvider) // need to generate AppWorkspaces only
		blobberAppPackageFS := parser.PackageFS{
			Path: BlobberAppFQN,
			FS:   blobberSchemaFS,
		}
		return apps.BuiltInAppDef{
			AppQName: istructs.AppQName_sys_blobber,
			Packages: []parser.PackageFS{sysPackageFS, blobberAppPackageFS},
			AppDeploymentDescriptor: cluster.AppDeploymentDescriptor{
				PartsCount:     DefDeploymentPartsCount,
				EnginePoolSize: DefDeploymentEnginePoolSize,
			},
		}
	}
}
