/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package vvm

import (
	"context"

	"github.com/voedger/voedger/pkg/istructs"
)

func (srv *AppPartsCtlPipelineService) Prepare(_ interface{}) error {
	return srv.IAppPartitionsController.Prepare()
}

func (srv *AppPartsCtlPipelineService) Run(ctx context.Context) {
	srv.IAppPartitionsController.Run(ctx)
}

func (srv *AppPartsCtlPipelineService) Stop() {}

type implINumAppPartitionsSource struct {
	defs []BuiltInAppPackages
}

func (i *implINumAppPartitionsSource) NumAppPartitions(appQName istructs.AppQName) istructs.NumAppPartitions {
	for _, def := range i.defs {
		if def.Name == appQName {
			return def.NumParts
		}
	}
	// notest
	panic("not found")
}
