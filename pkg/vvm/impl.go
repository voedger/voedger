/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package vvm

import (
	"context"
)

func (srv *AppPartsCtlPipelineService) Prepare(_ interface{}) error {
	return srv.IAppPartitionsController.Prepare()
}

func (srv *AppPartsCtlPipelineService) Run(ctx context.Context) {
	srv.IAppPartitionsController.Run(ctx)
}

func (srv *AppPartsCtlPipelineService) Stop() {}
