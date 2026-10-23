/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package checkpoints

import (
	"context"
	"fmt"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func (r *initialRecovery) InitializeCheckpoints(ctx context.Context, appQName appdef.AppQName,
	numPartitions istructs.NumAppPartitions) (deployablePartitions []istructs.PartitionID, failedPartitions map[istructs.PartitionID]error, err error) {
	app, err := r.appStructs.BuiltIn(appQName)
	if err != nil {
		return nil, nil, err
	}
	logCtx := logger.WithContextAttrs(ctx, map[string]any{
		logger.LogAttr_VApp:      sys.VApp_SysVoedger,
		logger.LogAttr_Extension: "sys._Recovery",
		"app":                    appQName,
	})
	logger.InfoCtx(logCtx, "bootstrap.initial_recovery.start", fmt.Sprintf("partitions %d, workers %d", numPartitions, workerLimit))

	results := make([]partitionResult, numPartitions)
	jobs := make(chan istructs.PartitionID)
	var workers sync.WaitGroup
	for range min(workerLimit, int(numPartitions)) {
		workers.Go(func() {
			for partitionID := range jobs {
				if ctx.Err() != nil {
					return
				}
				partCtx := logger.WithContextAttrs(logCtx, map[string]any{"partid": partitionID})
				alreadyInited, err := r.initializeCheckpoints(partCtx, app, partitionID)
				// Each job owns one result slot; readers wait until every worker joins.
				results[partitionID] = partitionResult{alreadyInited: alreadyInited, err: err}
			}
		})
	}
schedule:
	for id := range numPartitions {
		select {
		case jobs <- istructs.PartitionID(id):
		case <-ctx.Done():
			break schedule
		}
	}
	close(jobs)
	workers.Wait()
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	deployablePartitions = make([]istructs.PartitionID, 0, numPartitions)
	failedPartitions = make(map[istructs.PartitionID]error)
	newlyInited, alreadyInited := 0, 0
	for partitionID, partitionInitResult := range results {
		partitionID := istructs.PartitionID(partitionID) // nolint G115 bounded by numPartitions
		if partitionInitResult.err != nil {
			failedPartitions[partitionID] = partitionInitResult.err
			continue
		}
		deployablePartitions = append(deployablePartitions, partitionID)
		if partitionInitResult.alreadyInited {
			alreadyInited++
		} else {
			newlyInited++
		}
	}
	logger.InfoCtx(logCtx, "bootstrap.initial_recovery.complete",
		fmt.Sprintf("newly initialized %d, already initialized %d, failed %d", newlyInited, alreadyInited, len(failedPartitions)))
	return deployablePartitions, failedPartitions, nil
}

func (r *initialRecovery) initializeCheckpoints(ctx context.Context, app istructs.IAppStructs, partitionID istructs.PartitionID) (skipped bool, err error) {
	appID := app.ClusterAppID()
	if _, exists, err := r.storage.GetPartitionCheckpoint(appID, partitionID); err != nil {
		return false, fmt.Errorf("get partition checkpoint: %w", err)
	} else if exists {
		// safe to not to look on workspace checkpoint because partition checkpoint is stored after workspace checkpoints
		return true, nil
	}
	partitionCheckpoint := PartitionCheckpoint{}
	workspaceCheckpoints := make(map[istructs.WSID]WorkspaceCheckpoint)
	err = app.Events().ReadPLog(ctx, partitionID, istructs.FirstOffset, istructs.ReadToTheEnd,
		func(offset istructs.Offset, event istructs.IPLogEvent) error {
			defer event.Release()
			if err := ctx.Err(); err != nil {
				return err
			}
			partitionCheckpoint.LastPLogOffset = offset
			wsid := event.Workspace()
			workspaceCheckpoint := workspaceCheckpoints[wsid]
			if canAdvanceWorkspaceCheckpoint(event) {
				workspaceCheckpoint.WLogIDBearingOffset = event.WLogOffset()
			}
			workspaceCheckpoints[wsid] = workspaceCheckpoint
			return nil
		})
	if err != nil {
		return false, fmt.Errorf("read PLog: %w", err)
	}
	for wsid, checkpoint := range workspaceCheckpoints {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if err := r.storage.PutWorkspaceCheckpoint(appID, wsid, checkpoint); err != nil {
			return false, &WorkspaceCheckpointError{WSID: wsid, Err: err}
		}
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	// The partition marker must be the final write so retries cannot skip an
	// incomplete set of workspace checkpoints.
	if err := r.storage.PutPartitionCheckpoint(appID, partitionID, partitionCheckpoint); err != nil {
		return false, fmt.Errorf("put partition checkpoint: %w", err)
	}
	logger.InfoCtx(ctx, "bootstrap.initial_recovery.partition.complete",
		fmt.Sprintf("lastPLogOffset %d, workspaceCheckpoints stored %d", partitionCheckpoint.LastPLogOffset, len(workspaceCheckpoints)))
	return false, nil
}

func (e *WorkspaceCheckpointError) Error() string {
	return fmt.Sprintf("put workspace checkpoint: %s", e.Err)
}

func (e *WorkspaceCheckpointError) Unwrap() error { return e.Err }
