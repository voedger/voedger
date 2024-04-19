/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl

import (
	"context"

	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

type appPartitionsController struct {
	parts       appparts.IAppPartitions
	builtInApps []BuiltInApp
}

func newAppPartitionsController(parts appparts.IAppPartitions, apps []BuiltInApp) (ctl IAppPartitionsController, cleanup func(), err error) {
	apc := appPartitionsController{parts: parts, builtInApps: apps}

	return &apc, func() {}, err
}

func (ctl *appPartitionsController) Prepare() (err error) {
	return err
}

func (ctl *appPartitionsController) Run(ctx context.Context) {
	for _, builtinApp := range ctl.builtInApps {
		ctl.parts.DeployApp(builtinApp.Name, builtinApp.Def, builtinApp.NumParts, builtinApp.EnginePoolSize)
		ids := make([]istructs.PartitionID, builtinApp.NumParts)
		for id := coreutils.NumAppPartitions(0); id < builtinApp.NumParts; id++ {
			ids[id] = istructs.PartitionID(id)
		}
		ctl.parts.DeployAppPartitions(builtinApp.Name, ids)
	}

	<-ctx.Done()
}
