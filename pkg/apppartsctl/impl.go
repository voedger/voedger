/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl

import (
	"context"

	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/istructs"
)

type appPartitionsController struct {
	parts appparts.IAppPartitions
	apps  []BuiltInApp
}

func newAppPartitionsController(parts appparts.IAppPartitions, apps []BuiltInApp) (ctl IAppPartitionsController, cleanup func(), err error) {
	apc := appPartitionsController{parts: parts, apps: apps}

	return &apc, func() {}, err
}

func (ctl *appPartitionsController) Prepare() (err error) {
	return err
}

func (ctl *appPartitionsController) Run(ctx context.Context) {

	for _, app := range ctl.apps {
		for id := 1; id < app.NumParts; id++ {
			ctl.parts.AddOrReplace(app.Name, istructs.PartitionID(id), app.Def, app.Engines)
		}
	}

	<-ctx.Done()
}
