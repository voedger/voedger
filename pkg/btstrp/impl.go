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

	sysToken, err := payloads.GetSystemPrincipalToken(itokens, istructs.AppQName_sys_cluster)
	if err != nil {
		return err
	}

	// check apps compatibility
	for _, app := range otherApps {
		wasDeployed, deployedNumPartitions, deployedNumAppWorkspaces, err := readPreviousAppDeployment(federation, app.Name, sysToken)
		if err != nil {
			// notest
			return err
		}

		if !wasDeployed {
			// not deployed, call c.cluster.DeployApp
			if err := deployApp(federation, app.Name, app.NumParts, sysToken); err != nil {
				return err
			}
			continue
		}

		// was deployed somewhen -> check app compatibility
		if app.NumParts != deployedNumPartitions {
			return fmt.Errorf("app %s declaring NumPartitions=%d but was previously deployed with NumPartitions=%d", app.Name, app.NumParts, deployedNumPartitions)
		}
		if app.NumAppWorkspaces != deployedNumAppWorkspaces {
			return fmt.Errorf("app %s declaring NumAppWorkspaces=%d but was previously deployed with NumAppWorksaces=%d", app.Name, app.NumAppWorkspaces, deployedNumAppWorkspaces)
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

func readPreviousAppDeployment(federation coreutils.IFederation, appQName istructs.AppQName, sysToken string) (wasDeployed bool, deployedNumPartitions istructs.NumAppPartitions, deployedNumAppWorkspaces istructs.NumAppWorkspaces, err error) {
	body := fmt.Sprintf(`{"args":{"AppQName":"%s"},"elements":[{"fields": ["NumPartitions", "NumAppWorkspaces"]}]}`, appQName)
	resp, err := federation.Func(fmt.Sprintf("api/%s/%d/q.cluster.QueryApp", istructs.AppQName_sys_cluster, clusterapp.ClusterAppPseudoWSID), body,
		coreutils.WithAuthorizeBy(sysToken))
	if err != nil {
		return false, 0, 0, err
	}

	if len(resp.Sections) == 0 {
		return false, 0, 0, nil
	}
	deployedNumPartitions = istructs.NumAppPartitions(resp.SectionRow()[0].(float64))
	deployedNumAppWorkspaces = istructs.NumAppWorkspaces(resp.SectionRow()[1].(float64))
	return true, deployedNumPartitions, deployedNumAppWorkspaces, nil
}

func deployApp(federation coreutils.IFederation, appQName istructs.AppQName, numPartitions istructs.NumAppPartitions, sysToken string) error {
	body := fmt.Sprintf(`{"args":{"AppQName":"%s","NumPartitions":%d}}`, appQName, numPartitions)
	_, err := federation.Func(fmt.Sprintf("api/%s/%d/c.cluster.DeployApp", istructs.AppQName_sys_cluster, clusterapp.ClusterAppPseudoWSID), body,
		coreutils.WithDiscardResponse(), coreutils.WithAuthorizeBy(sysToken))
	return err
}

func initClusterAppWS(asp istructs.IAppStructsProvider, timeFunc coreutils.TimeFunc) error {
	as, err := asp.AppStructs(istructs.AppQName_sys_cluster)
	if err == nil {
		_, err = cluster.InitAppWS(as, clusterapp.ClusterAppWSIDPartitionID, clusterapp.ClusterAppWSID, istructs.FirstOffset, istructs.FirstOffset,
			istructs.UnixMilli(timeFunc().UnixMilli()))
	}
	return err
}
