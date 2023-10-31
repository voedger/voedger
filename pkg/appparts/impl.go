/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"

	"github.com/voedger/voedger/pkg/istructs"
)

type appPartitions struct {
}

func newAppPartitions() (ap IAppPartitions, cleanup func(), err error) {
	return &appPartitions{}, cleanup, err
}

func (ap appPartitions) Prepare() (err error) {
	return err
}

func (ap appPartitions) Run(ctx context.Context) {
	<-ctx.Done()
}

type appPartitionsAPI struct {
	ap IAppPartitions
}

func newAppPartitionsAPI(ap IAppPartitions) (api IAppPartitionsAPI, err error) {
	return &appPartitionsAPI{ap}, err
}

func (api *appPartitionsAPI) Get(istructs.AppQName) IAppPartition {
	return nil
}
