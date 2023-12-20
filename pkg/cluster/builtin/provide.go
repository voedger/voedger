/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package builtin

import (
	"github.com/voedger/voedger/pkg/apppartsctl"
	"github.com/voedger/voedger/pkg/apps/sys/blobberapp"
	"github.com/voedger/voedger/pkg/apps/sys/registryapp"
	"github.com/voedger/voedger/pkg/apps/sys/routerapp"
	"github.com/voedger/voedger/pkg/istructs"
)

// Returns built-in cluster applications to deploy by application partition controller
func Apps() []apppartsctl.BuiltInApp {
	return []apppartsctl.BuiltInApp{
		{
			Name:           istructs.AppQName_sys_blobber,
			PartsCount:     blobberapp.PartsCount(),
			Def:            blobberapp.AppDef(),
			EnginePoolSize: blobberapp.EnginePoolSize(),
		},
		{
			Name:           istructs.AppQName_sys_registry,
			PartsCount:     registryapp.PartsCount(),
			Def:            registryapp.AppDef(),
			EnginePoolSize: registryapp.EnginePoolSize(),
		},
		{
			Name:           istructs.AppQName_sys_router,
			PartsCount:     routerapp.PartsCount(),
			Def:            routerapp.AppDef(),
			EnginePoolSize: routerapp.EnginePoolSize(),
		},
	}
}
