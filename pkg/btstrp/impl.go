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
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// is a SyncOp within VVM trunk
func Bootstrap(federation coreutils.IFederation, asp istructs.IAppStructsProvider, timeFunc coreutils.TimeFunc, appparts appparts.IAppPartitions,
	clusterApp ClusterBuiltInApp, otherApps []appparts.BuiltInApp, itokens itokens.ITokens) error {
	// initialize cluster app workspace, use app ws amount 0
	if err := initClusterAppWS(asp, timeFunc); err != nil {
		return err
	}

	// deploy single clusterApp partition 0
	appparts.DeployApp(istructs.AppQName_sys_cluster, clusterApp.Def, clusterapp.ClusterAppNumPartitions, clusterapp.ClusterAppNumEngines)
	appparts.DeployAppPartitions(istructs.AppQName_sys_cluster, []istructs.PartitionID{clusterapp.ClusterAppWSIDPartitionID})

	// note: *IBlobberAppStorage and *IRouterAppStorage are initlaized in bootstrap sync operator, see vvm/provide.go:provideBootstrapOperator()

	sysToken, err := payloads.GetSystemPrincipalToken(itokens, istructs.AppQName_sys_cluster)
	if err != nil {
		return err
	}

	// check apps compatibility
	for _, app := range otherApps {
		body := fmt.Sprintf(`{"args":{"AppQName":"%s","NumPartitions":%d,"NumAppWorkspaces":%d}}`, app.Name, app.NumParts, app.NumAppWorkspaces)
		_, err := federation.AdminFunc(fmt.Sprintf("api/%s/%d/c.cluster.DeployApp", istructs.AppQName_sys_cluster, clusterapp.ClusterAppPseudoWSID), body,
			coreutils.WithDiscardResponse(),
			coreutils.WithAuthorizeBy(sysToken),
			coreutils.WithRetryOnAnyError(retryOnHTTPErrorTimeout, retryOnHTTPErrorDelay),
		)
		if err != nil {
			panic(fmt.Sprintf("failed to deploy app: %s", err.Error()))
		}
	}

	// appparts: deploy app and its partitions
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
