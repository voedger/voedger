/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

const AppPartitionBorrowRetryDelay = 50 * time.Millisecond

// NullSyncActualizerFactory should be used in test only
var NullSyncActualizerFactory SyncActualizerFactory = func(istructs.IAppStructs, istructs.PartitionID) pipeline.ISyncOperator {
	return &pipeline.NOOP{}
}

// NullExtensionEngineFactories should be used in test only
var NullExtensionEngineFactories iextengine.ExtensionEngineFactories = iextengine.ExtensionEngineFactories{
	appdef.ExtensionEngineKind_BuiltIn: iextengine.NullExtensionEngineFactory,
	appdef.ExtensionEngineKind_WASM:    iextengine.NullExtensionEngineFactory,
}

// NullActualizerRunner should be used in test only
var NullActualizerRunner nullActualizerRunner = nullActualizerRunner{}

// NullSchedulerRunner should be used in test only
var NullSchedulerRunner nullSchedulerRunner = nullSchedulerRunner{}
