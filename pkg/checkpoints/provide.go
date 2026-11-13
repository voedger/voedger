/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package checkpoints

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
)

// NewApplyRecoveryCheckpointProjector registers one asynchronous projector for both recovery checkpoints.
func NewApplyRecoveryCheckpointProjector(resources istructsmem.IStatelessResources, storage IRecoveryCheckpointStorage) {
	resources.AddProjectors(appdef.SysPackagePath,
		istructs.Projector{
			Name: QNameApplyRecoveryCheckpoint,
			Func: ApplyRecoveryCheckpoint(storage),
		},
	)
}

// NewInitialRecovery constructs the bootstrap checkpoint initializer.
func NewInitialRecovery(appStructs istructs.IAppStructsProvider, storage IRecoveryCheckpointStorage) IInitialRecovery {
	return &initialRecovery{appStructs: appStructs, storage: storage}
}
