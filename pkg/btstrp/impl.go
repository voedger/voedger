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
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/builtin"
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

	// // idem

	// // check apps compatibility
	// for _, app := range otherApps {
	// 	as, err := asp.AppStructs(app.AppQName)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	kb := as.ViewRecords().KeyBuilder(cluster.QNameViewDeployedApps)
	// 	kb.PutString(cluster.Field_AppQName, as.AppQName().String())
	// 	kb.PutInt32(cluster.Field_ClusterAppID, int32(istructs.ClusterApps[istructs.AppQName_sys_cluster]))
	// 	v, err := as.ViewRecords().Get(clusterAppWSID, kb)
	// 	if err != nil {
	// 		if errors.Is(err, istructsmem.ErrRecordNotFound) {
	// 			continue
	// 		}
	// 		// notest
	// 		return err
	// 	}
	// 	checked := false
	// 	deployEventWLogOffset := istructs.Offset(v.AsInt64(cluster.Field_DeployEventWLogOffset))
	// 	err = as.Events().ReadWLog(context.TODO(), clusterAppWSID, deployEventWLogOffset, 1, func(_ istructs.Offset, event istructs.IWLogEvent) (err error) {
	// 		deployedNumPartitions := istructs.NumAppPartitions(event.ArgumentObject().AsInt32(cluster.Field_NumPartitions))
	// 		deployedNumAppWorkspaces := istructs.NumAppWorkspaces(event.ArgumentObject().AsInt32(cluster.Field_NumAppWorkspaces))
	// 		if app.NumParts != deployedNumPartitions {
	// 			return fmt.Errorf("app %s declaring NumPartitions=%d but was previously deployed with NumPartitions=%d", app.AppQName, app.NumParts, deployedNumPartitions)
	// 		}
	// 		if app.NumAppWorkspaces != deployedNumAppWorkspaces {
	// 			return fmt.Errorf("app %s declaring NumAppWorkspaces=%d but was previously deployed with NumAppWorksaces=%d", app.AppQName, app.NumAppWorkspaces, deployedNumAppWorkspaces)
	// 		}
	// 		checked = true
	// 		return nil
	// 	})
	// 	if err != nil {
	// 		return err
	// 	}

	// 	if !checked {
	// 		return fmt.Errorf("failed to check %s app compatibility: event is not found by view.cluster.DeployedApps.DeployEventWLogOffset=%d", app.AppQName, deployEventWLogOffset)
	// 	}
	// }

	// // deploy apps
	// var x apppartsctl.BuiltInApp
	// for _, app := range otherApps {
	// 	appparts.DeployApp(app.AppQName)
	// }

	// return nil
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

func FindEventByODocOrORecordID(s istructs.IState, id istructs.RecordID) (value istructs.IStateValue, err error) {
	skbViewRecordsRegistry, err := s.KeyBuilder(state.View, builtin.QNameViewRecordsRegistry)
	if err != nil {
		return
	}
	skbViewRecordsRegistry.PutInt64(builtin.Field_IDHi, int64(builtin.CrackID(id)))
	skbViewRecordsRegistry.PutRecordID(builtin.Field_ID, id)
	svViewRecordsRegistry, err := s.MustExist(skbViewRecordsRegistry)
	if err != nil {
		return
	}

	skbWlog, err := s.KeyBuilder(state.WLog, state.WLog)
	if err != nil {
		return
	}
	skbWlog.PutInt64(state.Field_Offset, svViewRecordsRegistry.AsInt64(builtin.Field_WLogOffset))
	return s.MustExist(skbWlog)
}
