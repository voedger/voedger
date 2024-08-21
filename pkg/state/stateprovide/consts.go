/*
  - Copyright (c) 2024-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package stateprovide

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/state"
)

const (
	S_GET       = 1
	S_GET_BATCH = 2
	S_READ      = 4
	S_INSERT    = 8
	S_UPDATE    = 16
)

const (
	queryProcessorStateMaxIntents = 2 // 1 for Result, 1 for Response
)

var emptyApplyBatchItem = state.ApplyBatchItem{}

var (
	qNameCDocWorkspaceDescriptor = appdef.NewQName(appdef.SysPackage, "WorkspaceDescriptor")
)
