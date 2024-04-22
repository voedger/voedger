/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/parser"
)

func Provide(cfg *istructsmem.AppConfigType) parser.PackageFS {
	cfg.Resources.Add(istructsmem.NewQueryFunction(appdef.NewQName(ClusterPackage, "QueryApp"), provideExecQueryApp(cfg.AppDef)))
	cfg.Resources.Add(istructsmem.NewCommandFunction(appdef.NewQName(ClusterPackage, "DeployApp"), func(args istructs.ExecCommandArgs) (err error) { return nil }))
	cfg.AddSyncProjectors(istructs.Projector{
		Name: appdef.NewQName(ClusterPackage, "ApplyDeployApp"),
		Func: applyDeployApp,
	})
	return parser.PackageFS{
		Path: ClusterPackageFQN,
		FS:   schemaFS,
	}
}
