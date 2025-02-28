/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */
package vvm

import (
	"net"
	"testing/fstest"

	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/sys/smtp"
	"github.com/voedger/voedger/pkg/sys/sysprovide"
	builtinapps "github.com/voedger/voedger/pkg/vvm/builtin"
	"github.com/voedger/voedger/pkg/vvm/builtin/clusterapp"
	"github.com/voedger/voedger/pkg/vvm/builtin/registryapp"
)

// nolint:revive
func GetTestVVMCfg(ip net.IP) *VVMConfig {
	vvmCfg := NewVVMDefaultConfig()
	vvmCfg.VVMPort = 0
	vvmCfg.IP = ip
	vvmCfg.MetricsServicePort = 0
	vvmCfg.Time = coreutils.MockTime
	vvmCfg.VVMAppsBuilder.Add(istructs.AppQName_test1_app1, func(apis builtinapps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) builtinapps.Def {
		sysPackageFS := sysprovide.Provide(cfg)
		return builtinapps.Def{
			AppDeploymentDescriptor: appparts.AppDeploymentDescriptor{
				NumParts:         10,
				EnginePoolSize:   appparts.PoolSize(10, 10, 20, 10),
				NumAppWorkspaces: istructs.DefaultNumAppWorkspaces,
			},
			AppQName: istructs.AppQName_test1_app1,
			Packages: []parser.PackageFS{{
				Path: "github.com/voedger/voedger/pkg/app1",
				FS: fstest.MapFS{
					"app.vsql": &fstest.MapFile{
						Data: []byte(`
							APPLICATION app1();

							ALTERABLE WORKSPACE test_wsWS (

								DESCRIPTOR test_ws (
									IntFld int32 NOT NULL,
									StrFld varchar(1024)
								);
							);`),
					},
				},
			}, sysPackageFS},
		}
	})
	vvmCfg.VVMAppsBuilder.Add(istructs.AppQName_sys_registry, registryapp.Provide(smtp.Cfg{}, 10))
	vvmCfg.VVMAppsBuilder.Add(istructs.AppQName_sys_cluster, clusterapp.Provide())

	return &vvmCfg
}
