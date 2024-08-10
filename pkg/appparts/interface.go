/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"
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
		partsCount istructs.NumAppPartitions, numEngines [ProcessorKind_Count]int, numAppWorkspaces istructs.NumAppWorkspaces)

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
}

// dependency cycle: func requires IAppPartitions, provider of IAppPartitions requires already filled AppConfigsType -> impossible to provide AppConfigsType because we're filling it now
// TODO: eliminate this workaround
// type BuiltInAppsDeploymentDescriptors map[appdef.AppQName]AppDeploymentDescriptor

// Processor runner.
//
// Used by application partitions to run actualizers and schedulers
type IProcessorRunner interface {
	// Sets application partitions.
	//
	// Should be called before any other method.
	SetAppPartitions(IAppPartitions)

	// Creates and runs new processor for specified application partition
	NewAndRun(context.Context, appdef.AppQName, istructs.PartitionID, appdef.QName)
}
