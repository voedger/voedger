/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

type SyncActualizerFactory = func(istructs.IAppStructs, istructs.PartitionID) pipeline.ISyncOperator

// func New(structs istructs.IAppStructsProvider) (ap IAppPartitions, cleanup func(), err error) {
// 	return newAppPartitions(structs, nil)
// }

func New(structs istructs.IAppStructsProvider) (ap IAppPartitions, cleanup func(), err error) {
	return NewWithActualizer(structs, func(is istructs.IAppStructs, pi istructs.PartitionID) pipeline.ISyncOperator {
		return &pipeline.NOOP{}
	})
}

func NewWithActualizer(structs istructs.IAppStructsProvider, actualizer SyncActualizerFactory) (ap IAppPartitions, cleanup func(), err error) {
	return newAppPartitions(structs, actualizer)
}
