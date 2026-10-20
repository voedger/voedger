/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package checkpoints

import (
	"context"

	"github.com/voedger/voedger/pkg/appdef"
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

// IInitialRecovery initializes persistent recovery checkpoints before deployment.
type IInitialRecovery interface {
	// InitializeCheckpoints returns deployable partition IDs, errors keyed by failed
	// partition ID, and a fatal application setup or context error. When the fatal
	// error is nil, the IDs and error-map keys are disjoint and cover 0..numPartitions-1.
	// An empty deployable slice is valid when every partition fails. A non-nil fatal
	// error prohibits deployment from any partial results. A present partition
	// checkpoint, including zero, marks completion and requires no reconstruction.
	InitializeCheckpoints(ctx context.Context, appQName appdef.AppQName, numPartitions istructs.NumAppPartitions) (
		partitionIDsWithInitedCheckpoints []istructs.PartitionID,
		partitionIDsWithFailedCheckpoints map[istructs.PartitionID]error, // error could be about both partition and workspace checkpoints
		err error,
	)
}
