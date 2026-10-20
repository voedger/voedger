/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package checkpoints

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

type initialRecovery struct {
	appStructs istructs.IAppStructsProvider
	storage    IRecoveryCheckpointStorage
}

// WorkspaceCheckpointError identifies the workspace whose checkpoint write failed.
type WorkspaceCheckpointError struct {
	WSID istructs.WSID
	Err  error
}

func (e *WorkspaceCheckpointError) Error() string {
	return fmt.Sprintf("put workspace checkpoint: %s", e.Err)
}

func (e *WorkspaceCheckpointError) Unwrap() error { return e.Err }

// NewInitialRecovery constructs the bootstrap checkpoint initializer.
func NewInitialRecovery(appStructs istructs.IAppStructsProvider, storage IRecoveryCheckpointStorage) IInitialRecovery {
	return &initialRecovery{appStructs: appStructs, storage: storage}
}

func (r *initialRecovery) InitializeCheckpoints(ctx context.Context, appQName appdef.AppQName,
	numPartitions istructs.NumAppPartitions) ([]istructs.PartitionID, map[istructs.PartitionID]error, error) {
	workerLimit := runtime.GOMAXPROCS(0)
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	app, err := r.appStructs.BuiltIn(appQName)
	if contextErr := ctx.Err(); contextErr != nil {
		return nil, nil, contextErr
	}
	if err != nil {
		return nil, nil, fmt.Errorf("get application structures: %w", err)
	}
	logCtx := logger.WithContextAttrs(ctx, map[string]any{
		logger.LogAttr_VApp:      sys.VApp_SysVoedger,
		logger.LogAttr_Extension: "sys._Recovery",
		"app":                    appQName,
	})
	logger.InfoCtx(logCtx, "bootstrap.initial_recovery.start", fmt.Sprintf("partitions %d, workers %d", numPartitions, workerLimit))

	type partitionResult struct {
		completed bool
		skipped   bool
		err       error
	}
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
				skipped, err := r.initializePartition(partCtx, app, partitionID)
				// Each job owns one result slot; readers wait until every worker joins.
				results[partitionID] = partitionResult{completed: true, skipped: skipped, err: err}
			}
		})
	}
schedule:
	for id := range numPartitions {
		if ctx.Err() != nil {
			break
		}
		select {
		case jobs <- istructs.PartitionID(id):
		case <-ctx.Done():
			break schedule
		}
	}
	close(jobs)
	workers.Wait()

	deployable := make([]istructs.PartitionID, 0, numPartitions)
	failed := make(map[istructs.PartitionID]error)
	fatalErr := ctx.Err()
	initialized, skipped := 0, 0
	for id, result := range results {
		if !result.completed {
			continue
		}
		partitionID := istructs.PartitionID(id) // nolint G115 bounded by numPartitions
		if result.err != nil {
			// Context cancellation is fatal; retain other partition failures for logging.
			if fatalErr == nil || !errors.Is(result.err, fatalErr) {
				failed[partitionID] = result.err
			}
			continue
		}
		deployable = append(deployable, partitionID)
		if result.skipped {
			skipped++
		} else {
			initialized++
		}
	}
	if fatalErr != nil {
		return deployable, failed, fatalErr
	}
	logger.InfoCtx(logCtx, "bootstrap.initial_recovery.complete",
		fmt.Sprintf("initialized %d, skipped %d, failed %d", initialized, skipped, len(failed)))
	return deployable, failed, nil
}

func (r *initialRecovery) initializePartition(ctx context.Context, app istructs.IAppStructs, partitionID istructs.PartitionID) (skipped bool, err error) {
	appID := app.ClusterAppID()
	if _, exists, err := r.storage.GetPartitionCheckpoint(appID, partitionID); err != nil {
		return false, fmt.Errorf("get partition checkpoint: %w", err)
	} else if exists {
		return true, nil
	}
	partition := PartitionCheckpoint{}
	workspaces := make(map[istructs.WSID]WorkspaceCheckpoint)
	err = app.Events().ReadPLog(ctx, partitionID, istructs.FirstOffset, istructs.ReadToTheEnd,
		func(offset istructs.Offset, event istructs.IPLogEvent) error {
			defer event.Release()
			if err := ctx.Err(); err != nil {
				return err
			}
			partition.LastPLogOffset = offset
			wsid := event.Workspace()
			checkpoint := workspaces[wsid]
			if IsIDBearingEvent(event) {
				checkpoint.WLogIDBearingOffset = event.WLogOffset()
			}
			workspaces[wsid] = checkpoint
			return nil
		})
	if err != nil {
		return false, fmt.Errorf("read PLog: %w", err)
	}
	for wsid, checkpoint := range workspaces {
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
	if err := r.storage.PutPartitionCheckpoint(appID, partitionID, partition); err != nil {
		return false, fmt.Errorf("put partition checkpoint: %w", err)
	}
	logger.InfoCtx(ctx, "bootstrap.initial_recovery.partition.complete",
		fmt.Sprintf("lastPLogOffset %d, workspaceCheckpoints %d", partition.LastPLogOffset, len(workspaces)))
	return false, nil
}
