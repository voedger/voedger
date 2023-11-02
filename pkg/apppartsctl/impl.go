/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl

import (
	"context"

	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
)

type appPartitionsController struct {
	storages istorage.IAppStorageProvider
	apps     []BuiltInApp
	parts    appparts.IAppPartitions
}

func newAppPartitionsController(storages istorage.IAppStorageProvider, apps []BuiltInApp) (ctl IAppPartitionsController, cleanup func(), err error) {
	apc := appPartitionsController{storages: storages, apps: apps}

	apc.parts, cleanup, err = appparts.New(storages)
	if err != nil {
		return nil, nil, err
	}

	return &apc, cleanup, err
}

func (ctl *appPartitionsController) Prepare() (err error) {
	return err
}

func (ctl *appPartitionsController) Run(ctx context.Context) {

	for _, app := range ctl.apps {
		for id := 1; id < app.NumParts; id++ {
			ctl.parts.AddOrReplace(app.Name, istructs.PartitionID(id), app.Def)
		}
	}

	<-ctx.Done()
}
