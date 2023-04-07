/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ihttp

import (
	"context"
	"io/fs"

	"github.com/untillpro/voedger/pkg/ibus"
	"github.com/untillpro/voedger/pkg/iservices"
	"github.com/untillpro/voedger/pkg/istructs"
)

type CLIParams struct {
	// AppStorage istorage.IAppStorage
	Port int
	// HttpsPort  int
}

// Proposed factory signature
type NewType func(params CLIParams, bus ibus.IBus) (intf IHTTPProcessor, cleanup func(), err error)

type IHTTPProcessor interface {
	iservices.IService
}

type IHTTPProcessorAPI interface {
	/*
		Static Content

		url structure:
		<cluster-domain>/static/<AppQName.owner>/<AppQName.name>/<StaticFolderQName.pkg>/<StaticFolderQName.entity>/<content-path>

		example:
		<cluster-domain>/static/sys/monitor/site/hello/index.html

		- nil fs means that Static Content should be removed
		- Same resource can be deployed multiple times
	*/

	DeployStaticContent(ctx context.Context, path string, fs fs.FS) (err error)

	ListeningPort(ctx context.Context) (port int, err error)

	/*
		App Partitions

		<cluster-domain>/api/<AppQName.owner>/<AppQName.name>/<wsid>/<{q,c}.funcQName>
		Usage: (SetAppPartitionsNumber ( DeployAppPartition | UndeployAppPartition )*  UndeployAllAppPartitions)*
	*/

	//--	SetAppPartitionsNumber(app istructs.AppQName, partNo istructs.PartitionID, numPartitions istructs.PartitionID) (err error)

	// ErrUnknownApplication
	DeployAppPartition(ctx context.Context, app istructs.AppQName, partNo istructs.PartitionID, commandHandler, queryHandler ibus.ISender) (err error)

	// ErrUnknownAppPartition
	//--	UndeployAppPartition(app istructs.AppQName, partNo istructs.PartitionID) (err error)

	// ErrUnknownApplication
	//--	UndeployAllAppPartitions(app istructs.AppQName)

	/*
		Dynamic Subresources

		<cluster-domain>/api/<AppQName.owner>/<AppQName.name>/<StaticFolderQName.pkg>/<StaticFolderQName.entity>/<DynamicSubResource.Path>
		<Alias.Domain>/<Alias.Path>

		Usage: (DeployDynamicSubresource ( DeployDynamicSubresourceAlias | UndeployDynamicSubresourceAlias )* UndeployDynamicSubresource)*
	*/

	//--	DeployDynamicSubresource(app istructs.AppQName, path string, queryHandler ibus.ISender, Aliases []Alias) (err error)

	// ErrUnknownDynamicSubresource
	//--	DeployDynamicSubresourceAlias(app istructs.AppQName, path string, aliasDomain string, aliasPath string) (err error)

	// ErrUnknownDynamicSubresourceAlias
	//--	UndeployDynamicSubresourceAlias(app istructs.AppQName, path string, aliasDomain string, aliasPath string) (err error)

	// ErrUnknownDynamicSubresource
	//--	UndeployDynamicSubresource(app istructs.AppQName, path string) (err error)
}
