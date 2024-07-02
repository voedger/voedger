/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"context"
	"net/url"

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
	//
	// If application with the same name exists, then its definition will be updated.
	// наверное, сюда надо передать путь к app1/image, иначе движки не собрать
	DeployApp(name appdef.AppQName, extModuleURLs map[string]*url.URL, def appdef.IAppDef, partsCount istructs.NumAppPartitions, numEngines [ProcessorKind_Count]int)
	DeployBuiltInApp(name appdef.AppQName, def appdef.IAppDef, partsCount istructs.NumAppPartitions, numEngines [ProcessorKind_Count]int)
	// то, что раньше вызывало DeployApp, теперь должно вызывать DeployBuiltInApp

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

	DoSyncActualizer(ctx context.Context, work interface{}) (err error)

	// Invoke extension engine.
	Invoke(ctx context.Context, name appdef.QName, state istructs.IState, intents istructs.IIntents) error
}

// dependency cycle: func requires IAppPartitions, provider of IAppPartitions requires already filled AppConfigsType -> impossible to provide AppConfigsType because we're filling it now
// TODO: eliminate this workaround
// type BuiltInAppsDeploymentDescriptors map[appdef.AppQName]AppDeploymentDescriptor

// Application partition actualizers.
//
// Used by IAppPartitions to deploy and undeploy actualizers for application partitions
type IActualizers interface {
	// Assign application partitions manager.
	//
	// Should be called before any other method.
	SetAppPartitions(IAppPartitions)

	// Deploys actualizers for specified application partition.
	//
	// Should start new actualizers and stop removed actualizers
	DeployPartition(appdef.AppQName, istructs.PartitionID) error

	// Undeploy actualizers for specified application partition.
	//
	// Should stop all partition actualizers
	UndeployPartition(appdef.AppQName, istructs.PartitionID)
}
