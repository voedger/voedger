/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package btstrp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/router"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/workspace"
	coreutils "github.com/voedger/voedger/pkg/utils"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

// is a SyncOp within VVM trunk
func Bootstrap(ctx context.Context, bus ibus.IBus, asp istructs.IAppStructsProvider, timeFunc coreutils.TimeFunc, appparts appparts.IAppPartitions, clusterApp ClusterBuiltInApp, otherApps []cluster.BuiltInApp) error {
	// initialize cluster app workspace, use app ws amount 0
	if err := initClusterAppWS(asp, timeFunc); err != nil {
		return err
	}

	// deploy single clusterApp partition 0
	appparts.DeployApp(istructs.AppQName_sys_cluster, clusterApp.Def, clusterAppNumPartitions, clusterAppNumEngines)
	appparts.DeployAppPartitions(istructs.AppQName_sys_cluster, []istructs.PartitionID{clusterAppWSIDPartitionID})

	// check apps compatibility
	for _, app := range otherApps {
		wasDeployed, deployedNumPartitions, deployedNumAppWorkspaces, err := readPreviousAppDeployment(ctx, bus, app.Name)
		if err != nil {
			// notest
			return err
		}

		if !wasDeployed {
			// not deployed, call c.cluster.DeployApp
			if err := deployApp(ctx, bus, app); err != nil {
				return err
			}
			return nil
		}

		// was deployed somewhen -> check app compatibility
		if app.NumParts != istructs.NumAppPartitions(deployedNumPartitions) {
			return fmt.Errorf("app %s declaring NumPartitions=%d but was previously deployed with NumPartitions=%d", app.Name, app.NumParts, deployedNumPartitions)
		}
		if app.NumAppWorkspaces != istructs.NumAppWorkspaces(deployedNumAppWorkspaces) {
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

func readPreviousAppDeployment(ctx context.Context, bus ibus.IBus, appQName istructs.AppQName) (wasDeployed bool, deployedNumPartitions int, deployedNumAppWorkspaces int, err error) {
	queryAppBusRequest := ibus.Request{
		Method:          ibus.HTTPMethodPOST,
		WSID:            int64(clusterAppWSID),
		PartitionNumber: int(clusterAppWSIDPartitionID),
		Resource:        "q.cluster.QueryApp",
		AppQName:        istructs.AppQName_sys_cluster.String(),
		Body:            []byte(fmt.Sprintf(`{"args":{"AppQName":"%s"},"elements":[{"fields": ["NumPartitions", "NumAppWorkspaces"]}]}`, appQName)),
	}
	_, sections, secErr, err := bus.SendRequest2(ctx, queryAppBusRequest, ibus.DefaultTimeout)
	if err != nil {
		// notest
		return false, 0, 0, err
	}
	var sec ibus.ISection
	defer func() {
		router.DiscardSection(sec, ctx)
		for sec := range sections {
			router.DiscardSection(sec, ctx)
		}
	}()
	count := 0
	for sec = range sections {
		arrSec, ok := sec.(ibus.IArraySection)
		if !ok {
			// notest
			err = errors.New("non-array section is returned")
			return
		}
		defer func() {
			for _, ok := arrSec.Next(ctx); ok; arrSec.Next(ctx) {
			}
		}()

		for elemBytes, ok := arrSec.Next(ctx); ok; elemBytes, ok = arrSec.Next(ctx) {
			switch count {
			case 0:
				if deployedNumPartitions, err = strconv.Atoi(string(elemBytes)); err != nil {
					// notest
					return
				}
				count++
			case 1:
				if deployedNumAppWorkspaces, err = strconv.Atoi(string(elemBytes)); err != nil {
					// notest
					return
				}
				count++
			default:
				// notest
				err = errors.New("unexpected section element received on reading q.cluster.QueryApp reply: " + string(elemBytes))
				return
			}
		}
	}
	err = *secErr
	wasDeployed = count == 2
	return
}

func deployApp(ctx context.Context, bus ibus.IBus, builtinApp cluster.BuiltInApp) error {
	req := ibus.Request{
		Method:          ibus.HTTPMethodPOST,
		WSID:            int64(clusterAppWSID),
		PartitionNumber: int(clusterAppWSIDPartitionID),
		Resource:        "c.cluster.DeployApp",
		AppQName:        istructs.AppQName_sys_cluster.String(),
		Body: []byte(fmt.Sprintf(`{"args":["AppQName":"%s","NumPartitions":%d,"NumAppWorkspaces":%d]}`, builtinApp.Name,
			builtinApp.NumParts, builtinApp.NumAppWorkspaces)),
	}
	resp, _, _, err := bus.SendRequest2(ctx, req, ibus.DefaultTimeout)
	if err != nil {
		// notest
		return err
	}
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	m := map[string]interface{}{}
	if err = json.Unmarshal(resp.Data, &m); err != nil {
		// notest
		return err
	}
	sysErrorMap := m["sys.Error"].(map[string]interface{})
	return coreutils.SysError{
		HTTPStatus: int(sysErrorMap["HTTPStatus"].(float64)),
		Message:    sysErrorMap["Message"].(string),
	}
}

func initClusterAppWS(asp istructs.IAppStructsProvider, timeFunc coreutils.TimeFunc) error {
	as, err := asp.AppStructs(istructs.AppQName_sys_cluster)
	if err != nil {
		return err
	}
	if err := initAppWS(as, clusterAppWSIDPartitionID, clusterAppWSID, istructs.FirstOffset, istructs.FirstOffset, istructs.UnixMilli(timeFunc().UnixMilli())); err != nil {
		return err
	}

	return nil
}

func initAppWS(as istructs.IAppStructs, partitionID istructs.PartitionID, wsid istructs.WSID, plogOffset, wlogOffset istructs.Offset, currentMillis istructs.UnixMilli) error {
	existingCDocWSDesc, err := as.Records().GetSingleton(wsid, authnz.QNameCDocWorkspaceDescriptor)
	if err != nil {
		return err
	}
	if existingCDocWSDesc.QName() != appdef.NullQName {
		logger.Verbose("app workspace", as.AppQName(), wsid-wsid.BaseWSID(), "(", wsid, ") inited already")
		return nil
	}

	grebp := istructs.GenericRawEventBuilderParams{
		HandlingPartition: partitionID,
		Workspace:         wsid,
		QName:             istructs.QNameCommandCUD,
		RegisteredAt:      currentMillis,
		PLogOffset:        plogOffset,
		WLogOffset:        wlogOffset,
	}
	reb := as.Events().GetSyncRawEventBuilder(
		istructs.SyncRawEventBuilderParams{
			GenericRawEventBuilderParams: grebp,
			SyncedAt:                     currentMillis,
		},
	)
	cdocWSDesc := reb.CUDBuilder().Create(authnz.QNameCDocWorkspaceDescriptor)
	cdocWSDesc.PutRecordID(appdef.SystemField_ID, 1)
	cdocWSDesc.PutString(authnz.Field_WSName, "appWS0")
	cdocWSDesc.PutQName(authnz.Field_WSKind, authnz.QNameCDoc_WorkspaceKind_AppWorkspace)
	cdocWSDesc.PutInt64(authnz.Field_CreatedAtMs, int64(currentMillis))
	cdocWSDesc.PutInt64(workspace.Field_InitCompletedAtMs, int64(currentMillis))
	rawEvent, err := reb.BuildRawEvent()
	if err != nil {
		return err
	}
	// ok to local IDGenerator here. Actual next record IDs will be determined on the partition recovery stage
	pLogEvent, err := as.Events().PutPlog(rawEvent, nil, istructsmem.NewIDGenerator())
	if err != nil {
		return err
	}
	pLogEvent.Release()
	logger.Verbose("app workspace", as.AppQName(), wsid-wsid.BaseWSID(), "(", wsid, ") initialized")
	return nil
}
