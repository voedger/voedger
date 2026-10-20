/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package checkpoints

import (
	"github.com/voedger/voedger/pkg/istructs"
)

// PartitionCheckpoint presence marks initial recovery complete, even at offset zero.
type PartitionCheckpoint struct {
	// LastPLogOffset is zero for an empty PLog. Zero is a non-validating
	// full-scan sentinel for runtime recovery.
	LastPLogOffset istructs.Offset `json:"lastPLogOffset"`
}

type WorkspaceCheckpoint struct {
	// WLogIDBearingOffset identifies a locally generated ID-bearing WLog event
	// from which recovery can scan without skipping the maximum allocated ID.
	// It need not identify the latest ID-bearing event.
	// Initial recovery stores zero when no such event exists, including sync-only history;
	// zero is a non-validating full-scan sentinel for runtime recovery.
	WLogIDBearingOffset istructs.Offset `json:"wlogIDBearingOffset"`
}

type initialRecovery struct {
	appStructs istructs.IAppStructsProvider
	storage    IRecoveryCheckpointStorage
}

// WorkspaceCheckpointError identifies the workspace whose checkpoint write failed.
type WorkspaceCheckpointError struct {
	WSID istructs.WSID
	Err  error
}

type partitionResult struct {
	alreadyInited bool
	err           error
}
