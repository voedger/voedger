/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package checkpoints

import (
	"runtime"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

const partitionCheckpointUpdateInterval istructs.Offset = 100

var (
	QNameApplyRecoveryCheckpoint = appdef.NewQName(appdef.SysPackage, "ApplyRecoveryCheckpoint")
	workerLimit                  = runtime.GOMAXPROCS(0)
)
