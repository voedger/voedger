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
	parts    appparts.IAppPartitions
}

func newAppPartitionsController(storages istorage.IAppStorageProvider, builtIn ...IApplication) (ctl IAppPartitionsController, cleanup func(), err error) {
	apc := appPartitionsController{storages: storages}

	apc.parts, cleanup, err = appparts.New()

	for _, app := range builtIn {
		apc.parts.AddApp(app.AppName(), app.AppDef())
		app.Partitions(func(id istructs.PartitionID) {
			apc.parts.AddPartition(app.AppName(), id)
		})
	}
	return &apc, cleanup, err
}

func (ap appPartitionsController) Prepare() (err error) {
	return err
}

func (ap appPartitionsController) Run(ctx context.Context) {
	<-ctx.Done()
}
