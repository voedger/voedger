/*
 * Copyright (c) 2023-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package vvm

import "context"

func (srv *AppPartsCtlPipelineService) Prepare(_ interface{}) error {
	return srv.IAppPartitionsController.Prepare()
}

func (srv *AppPartsCtlPipelineService) Run(ctx context.Context) {
	srv.IAppPartitionsController.Run(ctx)
}

func (srv *AppPartsCtlPipelineService) Stop() {}
