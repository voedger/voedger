/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ihttp

import (
	"context"
	"io/fs"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/iservices"
	"github.com/voedger/voedger/pkg/istructs"
)

// Proposed factory signature
type IHTTPProcessor interface {
	iservices.IService
	ListeningPort() int
	AddReverseProxyRoute(srcRegExp, dstRegExp string)
	SetReverseProxyRouteDefault(srcRegExp, dstRegExp string)
	AddAcmeDomain(domain string)
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

	//--	SetAppPartitionsNumber(app appdef.AppQName, partNo istructs.PartitionID, numPartitions istructs.PartitionID) (err error)

	// ErrUnknownApplication
	DeployAppPartition(app appdef.AppQName, partNo istructs.PartitionID, appPartitionRequestHandler bus.RequestHandler) (err error)
	// ErrUnknownAppPartition
	//--	UndeployAppPartition(app appdef.AppQName, partNo istructs.PartitionID) (err error)

	// ErrUnknownApplication
	//--	UndeployAllAppPartitions(app appdef.AppQName)
	DeployApp(app appdef.AppQName, numPartitions istructs.NumAppPartitions, numAppWS istructs.NumAppWorkspaces) (err error)
	UndeployAppPartition(app appdef.AppQName, partNo istructs.PartitionID) (err error)
	UndeployApp(app appdef.AppQName) (err error)
}

type ISender interface {
	// err.Error() must have QName format:
	//   var ErrBusTimeoutExpired = errors.New("coreutils.ErrSendTimeoutExpired")
	// NullHandler can be used as a reader
	Send(ctx context.Context, request interface{}, sectionsHandler SectionsHandlerType) (response interface{}, status Status, err error)
}
