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
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func Provide(cfg *istructsmem.AppConfigType, asp istructs.IAppStructsProvider, timeFunc coreutils.TimeFunc) parser.PackageFS {
	cfg.Resources.Add(istructsmem.NewQueryFunction(appdef.NewQName(ClusterPackage, "QueryApp"), provideExecQueryApp(asp)))
	cfg.Resources.Add(istructsmem.NewCommandFunction(appdef.NewQName(ClusterPackage, "DeployApp"), provideExecDeployApp(asp, timeFunc)))
	return parser.PackageFS{
		Path: ClusterPackageFQN,
		FS:   schemaFS,
	}
}
