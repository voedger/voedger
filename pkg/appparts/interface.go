/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"
	"iter"
	"net/url"

	"github.com/voedger/voedger/pkg/pipeline"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

// Application partitions manager.
//
// @ConcurrentAccess
type IAppPartitions interface {
	// Adds new application or update existing.
	//
	// partsCount - total partitions count for the application.
	// extensionModules is used for non-builtin apps only. Provide nil for others
	// numAppWorkspaces is used for non-builtin appse. Provide e.g. -1 for others
	//
	// If application with the same name exists, then its definition will be updated.
	DeployApp(name appdef.AppQName, extModuleURLs map[string]*url.URL, def appdef.IAppDef,
		partsCount istructs.NumAppPartitions, numEngines [ProcessorKind_Count]uint, numAppWorkspaces istructs.NumAppWorkspaces)

	// Deploys new partitions for specified application or update existing.
	//
	// If partition with the same app and id already exists, it will be updated.
	//
	// # Panics:
	// 	- if application not exists
	DeployAppPartitions(appName appdef.AppQName, partIDs []istructs.PartitionID)

	// Returns application definition.
	//
	// Returns nil and error if app not exists.
	AppDef(appdef.AppQName) (appdef.IAppDef, error)

	// Returns _total_ application partitions count.
	//
	// This is a configuration value for the application, independent of how many sections are currently deployed.
	//
	// Returns 0 and error if app not exists.
	AppPartsCount(appdef.AppQName) (istructs.NumAppPartitions, error)

	// Returns partition ID for specified workspace
	//
	// Returns error if app not exists.
	AppWorkspacePartitionID(appdef.AppQName, istructs.WSID) (istructs.PartitionID, error)

	// Borrows and returns a partition.
	//
	// If partition not exist, returns error.
	Borrow(appdef.AppQName, istructs.PartitionID, ProcessorKind) (IAppPartition, error)

	// Waits for partition to be available and borrows it.
	//
	// If partition not exist, returns error.
	WaitForBorrow(context.Context, appdef.AppQName, istructs.PartitionID, ProcessorKind) (IAppPartition, error)

	// Returns iterator for actualizers from deployed partitions.
	//
	// This method snapshots the current state of the partitions.
	// The snapshotted iterator are not updated when partitions are deployed or removed.
	WorkedActualizers(appdef.AppQName) iter.Seq2[istructs.PartitionID, []appdef.QName]

	// Returns iterator for schedulers from deployed partitions.
	//
	// This method snapshots the current state of the partitions.
	// The snapshotted iterator are not updated when partitions are deployed or removed.
	WorkedSchedulers(appdef.AppQName) iter.Seq2[istructs.PartitionID, map[appdef.QName][]istructs.WSID]

	// Upgrade application definition.
	//
	// This experimental method should be used for test purposes only.
	// Should be deprecated after application redeployment is implemented.
	UpgradeAppDef(appdef.AppQName, appdef.IAppDef)
}

// Application partition.
type IAppPartition interface {
	App() appdef.AppQName
	ID() istructs.PartitionID

	AppStructs() istructs.IAppStructs

	// Releases borrowed partition
	Release()

	DoSyncActualizer(ctx context.Context, work pipeline.IWorkpiece) (err error)

	// Invoke extension engine.
	Invoke(ctx context.Context, name appdef.QName, state istructs.IState, intents istructs.IIntents) error

	// Returns true if specified operation is allowed in specified workspace on specified resource for any of specified roles.
	//
	// If resource is any structure and operation is UPDATE, INSERT or SELECT, then if fields list specified, then result consider it,
	// else fields list is ignored.
	//
	// If some error in arguments, (ws or resource not found, operation is not applicable to resource, etc…) then error is returned.
	IsOperationAllowed(ws appdef.IWorkspace, op appdef.OperationKind, res appdef.QName, fld []appdef.FieldName, roles []appdef.QName) (bool, error)

	// Return is specified resource (command, query or structure) usage limit is exceeded.
	//
	// If resource usage is exceeded then returns name of first exceeded limit.
	IsLimitExceeded(resource appdef.QName, operation appdef.OperationKind, workspace istructs.WSID, remoteAddr string) (exceed bool, limit appdef.QName)
}

// Async actualizer runner.
//
// Used by application partitions to run actualizers
type IActualizerRunner interface {
	// Sets application partitions.
	//
	// Should be called before any other method.
	SetAppPartitions(IAppPartitions)

	// Creates and runs new async actualizer for specified application partition
	NewAndRun(context.Context, appdef.AppQName, istructs.PartitionID, appdef.QName)
}

// Scheduler runner.
//
// Used by application partitions to run schedulers
type ISchedulerRunner interface {
	// Sets application partitions.
	//
	// Should be called before any other method.
	SetAppPartitions(IAppPartitions)

	// Creates and runs new specified job scheduler for specified application partition and workspace
	NewAndRun(ctx context.Context, app appdef.AppQName, partition istructs.PartitionID, wsIdx istructs.AppWorkspaceNumber, wsid istructs.WSID, job appdef.QName)
}
