/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl

import (
	"context"

	"github.com/voedger/voedger/pkg/appparts"
)

type appPartitionsController struct {
	api appparts.IAppPartitionsAPI
}

func newAppPartitionsController(api appparts.IAppPartitionsAPI) (ctl IAppPartitionsController, cleanup func(), err error) {
	return &appPartitionsController{api: api}, cleanup, err
}

func (ap appPartitionsController) Prepare() (err error) {
	return err
}

func (ap appPartitionsController) Run(ctx context.Context) {
	<-ctx.Done()
}
