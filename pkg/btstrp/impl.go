/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package btstrp

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/apps/sys/clusterapp"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/iblobstoragestg"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/utils/federation"
	dbcertcache "github.com/voedger/voedger/pkg/vvm/db_cert_cache"
)

func Bootstrap(federation federation.IFederation, asp istructs.IAppStructsProvider, timeFunc coreutils.TimeFunc, appparts appparts.IAppPartitions,
	clusterApp ClusterBuiltInApp, otherApps []appparts.BuiltInApp, itokens itokens.ITokens, storageProvider istorage.IAppStorageProvider,
	blobberAppStoragePtr iblobstoragestg.BlobAppStoragePtr, routerAppStoragePtr dbcertcache.RouterAppStoragePtr) (err error) {

	// initialize cluster app workspace, use app ws amount 0
	if err := initClusterAppWS(asp, timeFunc); err != nil {
		return err
	}

	// Initialize AppStorageBlobber (* IAppStorage), AppStorageRouter (* IAppStorage)
	if *blobberAppStoragePtr, err = storageProvider.AppStorage(istructs.AppQName_sys_blobber); err != nil {
		// notest
		return err
	}
	if *routerAppStoragePtr, err = storageProvider.AppStorage(istructs.AppQName_sys_router); err != nil {
		// notest
		return err
	}

	// appparts: deploy single clusterApp partition
	appparts.DeployApp(istructs.AppQName_sys_cluster, clusterApp.Def, clusterapp.ClusterAppNumPartitions, clusterapp.ClusterAppNumEngines)
	appparts.DeployAppPartitions(istructs.AppQName_sys_cluster, []istructs.PartitionID{clusterapp.ClusterAppWSIDPartitionID})

	sysToken, err := payloads.GetSystemPrincipalToken(itokens, istructs.AppQName_sys_cluster)
	if err != nil {
		return err
	}

	// For each app in otherApps: check apps compatibility by calling c.cluster.DeployApp
	for _, app := range otherApps {
		// Use Admin Endpoint to send requests
		body := fmt.Sprintf(`{"args":{"AppQName":"%s","NumPartitions":%d,"NumAppWorkspaces":%d}}`, app.Name, app.NumParts, app.NumAppWorkspaces)
		_, err := federation.AdminFunc(fmt.Sprintf("api/%s/%d/c.cluster.DeployApp", istructs.AppQName_sys_cluster, clusterapp.ClusterAppPseudoWSID), body,
			coreutils.WithDiscardResponse(),
			coreutils.WithAuthorizeBy(sysToken),

			// here we expecting that the network could be not available on the VVM launch (e.g. balancer thinks the node is not up yet)
			coreutils.WithRetryOnAnyError(retryOnHTTPErrorTimeout, retryOnHTTPErrorDelay),
		)
		if err != nil {
			panic(fmt.Sprintf("failed to deploy app %s: %s", app.Name, err.Error()))
		}
	}

	// For each app builtInApps: deploy a builtin app
	for _, app := range otherApps {
		appparts.DeployApp(app.Name, app.Def, app.NumParts, app.EnginePoolSize)
		partitionIDs := make([]istructs.PartitionID, app.NumParts)
		for id := istructs.NumAppPartitions(0); id < app.NumParts; id++ {
			partitionIDs[id] = istructs.PartitionID(id)
		}
		appparts.DeployAppPartitions(app.Name, partitionIDs)
	}

	return nil
}

func initClusterAppWS(asp istructs.IAppStructsProvider, timeFunc coreutils.TimeFunc) error {
	as, err := asp.AppStructs(istructs.AppQName_sys_cluster)
	if err == nil {
		_, err = cluster.InitAppWS(as, clusterapp.ClusterAppWSIDPartitionID, clusterapp.ClusterAppWSID, istructs.FirstOffset, istructs.FirstOffset,
			istructs.UnixMilli(timeFunc().UnixMilli()))
	}
	return err
}
