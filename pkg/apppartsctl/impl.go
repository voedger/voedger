/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl

import (
	"context"
	"errors"

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

	apc.parts, cleanup, err = appparts.New(storages)
	if err != nil {
		return nil, nil, err
	}

	for _, app := range builtIn {
		err = errors.Join(err,
			apc.parts.AddApp(app.AppName(), app.AppDef()))
		app.Partitions(func(id istructs.PartitionID) {
			err = errors.Join(err,
				apc.parts.AddPartition(app.AppName(), id))
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
