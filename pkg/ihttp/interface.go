/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ihttp

import (
	"context"
	"io/fs"
	"net/http"

	"github.com/voedger/voedger/pkg/iservices"
	"github.com/voedger/voedger/pkg/istructs"
)

// Proposed factory signature
type IHTTPProcessor interface {
	iservices.IService
	ListeningPort() int
	HandlePath(resource string, prefix bool, handlerFunc func(http.ResponseWriter, *http.Request))
	AddReverseProxyRoute(srcRegExp, dstRegExp string)
	SetReverseProxyRouteDefault(srcRegExp, dstRegExp string)
	AddAcmeDomain(domain string)
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

	DeployStaticContent(path string, fs fs.FS)

	/*
		App Partitions

		<cluster-domain>/api/<AppQName.owner>/<AppQName.name>/<wsid>/<{q,c}.funcQName>
		Usage: (SetAppPartitionsNumber ( DeployAppPartition | UndeployAppPartition )*  UndeployAllAppPartitions)*
	*/

	//--	SetAppPartitionsNumber(app istructs.AppQName, partNo istructs.PartitionID, numPartitions istructs.PartitionID) (err error)

	// ErrUnknownApplication
	DeployAppPartition(app istructs.AppQName, partNo istructs.PartitionID, commandHandler, queryHandler ISender)

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
	AddReverseProxyRoute(srcRegExp, dstRegExp string)
	SetReverseProxyRouteDefault(srcRegExp, dstRegExp string)
	AddAcmeDomain(domain string)
}

type ISender interface {
	// err.Error() must have QName format:
	//   var ErrTimeoutExpired = errors.New("ibus.ErrTimeoutExpired")
	// NullHandler can be used as a reader
	Send(ctx context.Context, request interface{}, sectionsHandler SectionsHandlerType) (response interface{}, status Status, err error)
}
