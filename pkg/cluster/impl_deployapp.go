/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"fmt"
	"math"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
	"github.com/voedger/voedger/pkg/sys/uniques"
	"github.com/voedger/voedger/pkg/sys/workspace"
)

// wrong to use IAppPartitions to get total NumAppPartition because the app the cmd is called for is not deployed yet
func provideCmdDeployApp(asp istructs.IAppStructsProvider, time timeu.ITime, sidecarApps []appparts.SidecarApp) istructsmem.ExecCommandClosure {
	return func(args istructs.ExecCommandArgs) (err error) {
		appQNameStr := args.ArgumentObject.AsString(Field_AppQName)
		appQName, err := appdef.ParseAppQName(appQNameStr)
		if err != nil {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("failed to parse AppQName %s: %s", appQNameStr, err.Error()))
		}

		if appQName == istructs.AppQName_sys_cluster {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("%s app can not be deployed by c.cluster.DeployApp", istructs.AppQName_sys_cluster))
		}

		clusterAppStructs, err := asp.BuiltIn(istructs.AppQName_sys_cluster)
		if err != nil {
			// notest
			return err
		}
		wdocAppRecordID, err := uniques.GetRecordIDByUniqueCombination(args.WSID, qNameWDocApp, clusterAppStructs, map[string]interface{}{
			Field_AppQName: appQNameStr,
		})
		if err != nil {
			// notest
			return err
		}

		numAppWSInt := args.ArgumentObject.AsInt32(Field_NumAppWorkspaces)
		if numAppWSInt <= 0 || numAppWSInt > istructs.MaxNumAppWorkspaces {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("app workspaces number must be >0 and <%d", istructs.MaxNumAppWorkspaces))
		}
		numPartitionsInt := args.ArgumentObject.AsInt32(Field_NumPartitions)
		if numPartitionsInt < 0 || numPartitionsInt > math.MaxUint16 {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("app partitions number must be >0 and <%d", math.MaxUint16))
		}
		numAppWorkspacesToDeploy := istructs.NumAppWorkspaces(numAppWSInt)      // nolint G115 checked above
		numAppPartitionsToDeploy := istructs.NumAppPartitions(numPartitionsInt) // nolint G115 checked above
		if wdocAppRecordID != istructs.NullRecordID {
			kb, err := args.State.KeyBuilder(sys.Storage_Record, qNameWDocApp)
			if err != nil {
				// notest
				return err
			}
			kb.PutRecordID(sys.Storage_Record_Field_ID, wdocAppRecordID)
			appRec, err := args.State.MustExist(kb)
			if err != nil {
				// notest
				return err
			}
			numPartitionsDeployed := istructs.NumAppPartitions(appRec.AsInt32(Field_NumPartitions))       // nolint G115 checked above
			numAppWorkspacesDeployed := istructs.NumAppWorkspaces(appRec.AsInt32(Field_NumAppWorkspaces)) // nolint G115 checked above

			// Check application compatibility (409)
			if numPartitionsDeployed != numAppPartitionsToDeploy {
				return coreutils.NewHTTPErrorf(http.StatusConflict, fmt.Sprintf("%s: app %s declaring NumPartitions=%d but was previously deployed with NumPartitions=%d", ErrNumPartitionsChanged.Error(),
					appQName, numAppPartitionsToDeploy, numPartitionsDeployed))
			}
			if numAppWorkspacesDeployed != numAppWorkspacesToDeploy {
				return coreutils.NewHTTPErrorf(http.StatusConflict, fmt.Sprintf("%s: app %s declaring NumAppWorkspaces=%d but was previously deployed with NumAppWorkspaces=%d", ErrNumAppWorkspacesChanged.Error(),
					appQName, numAppWorkspacesToDeploy, numAppWorkspacesDeployed))
			}

			// idempotency: was deployed already and nothing changed -> do not initiaize app workspaces
			return nil
		}

		kb, err := args.State.KeyBuilder(sys.Storage_Record, qNameWDocApp)
		if err != nil {
			// notest
			return err
		}
		vb, err := args.Intents.NewValue(kb)
		if err != nil {
			// notest
			return err
		}

		vb.PutRecordID(appdef.SystemField_ID, 1)
		vb.PutString(Field_AppQName, appQNameStr)
		vb.PutInt32(Field_NumAppWorkspaces, int32(numAppWorkspacesToDeploy))
		vb.PutInt32(Field_NumPartitions, int32(numAppPartitionsToDeploy))

		// Create storage if not exists
		// Initialize appstructs data
		// note: for builtin apps that does nothing because IAppStructs is already initialized (including storage initialization) on VVM wiring
		// note: it is good that it is done here, not before return if nothing changed because we're want to initialize (i.e. create) keyspace here - that must be done once
		var as istructs.IAppStructs
		if sidecarApp, ok := isSidecarApp(appQName, sidecarApps); ok {
			as, err = asp.New(appQName, sidecarApp.Def, istructs.ClusterApps[appQName], sidecarApp.NumAppWorkspaces)
		} else {
			as, err = asp.BuiltIn(appQName)
		}
		if err != nil {
			// notest
			return fmt.Errorf("failed to get IAppStructs for %s: %w", appQName, err)
		}

		// Initialize app workspaces
		if _, err = InitAppWSes(as, numAppWorkspacesToDeploy, numAppPartitionsToDeploy, istructs.UnixMilli(time.Now().UnixMilli())); err != nil {
			// notest
			return fmt.Errorf("failed to deploy %s: %w", appQName, err)
		}
		logger.Info(fmt.Sprintf("app %s successfully deployed: NumPartitions=%d, NumAppWorkspaces=%d", appQName, numAppPartitionsToDeploy, numAppWorkspacesToDeploy))
		return nil
	}
}

func isSidecarApp(appQName appdef.AppQName, sidecarApps []appparts.SidecarApp) (sa appparts.SidecarApp, ok bool) {
	for _, app := range sidecarApps {
		if app.Name == appQName {
			return app, true
		}
	}
	return sa, false
}

// returns an array of inited AppWSIDs. Inited already -> AppWSID is not in the array. Need for testing only
func InitAppWSes(as istructs.IAppStructs, numAppWorkspaces istructs.NumAppWorkspaces, numAppPartitions istructs.NumAppPartitions, currentMillis istructs.UnixMilli) ([]istructs.WSID, error) {
	pLogOffsets := map[istructs.PartitionID]istructs.Offset{}
	wLogOffset := istructs.FirstOffset
	res := []istructs.WSID{}
	for wsNum := uint16(0); istructs.NumAppWorkspaces(wsNum) < numAppWorkspaces; wsNum++ {
		appWSID := istructs.NewWSID(istructs.CurrentClusterID(), istructs.WSID(wsNum)+istructs.FirstBaseAppWSID)
		partitionID := coreutils.AppPartitionID(appWSID, numAppPartitions)
		if _, ok := pLogOffsets[partitionID]; !ok {
			pLogOffsets[partitionID] = istructs.FirstOffset
		}
		inited, err := InitAppWS(as, partitionID, appWSID, pLogOffsets[partitionID], wLogOffset, currentMillis)
		if err != nil {
			// notest
			return nil, err
		}
		pLogOffsets[partitionID]++
		wLogOffset++
		if inited {
			res = append(res, appWSID)
		}
	}
	return res, nil
}

func InitAppWS(as istructs.IAppStructs, partitionID istructs.PartitionID, appWSID istructs.WSID, plogOffset, wlogOffset istructs.Offset, currentMillis istructs.UnixMilli) (inited bool, err error) {
	existingCDocWSDesc, err := as.Records().GetSingleton(appWSID, authnz.QNameCDocWorkspaceDescriptor)
	if err != nil {
		// notest
		return false, err
	}
	if existingCDocWSDesc.QName() != appdef.NullQName {
		logger.Verbose("app workspace", as.AppQName(), appWSID-appWSID.BaseWSID(), "(", appWSID, ") inited already")
		return false, nil
	}

	grebp := istructs.GenericRawEventBuilderParams{
		HandlingPartition: partitionID,
		Workspace:         appWSID,
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
		// notest
		return false, err
	}
	// ok to local IDGenerator here. Actual next record IDs will be determined on the partition recovery stage
	pLogEvent, err := as.Events().PutPlog(rawEvent, nil, istructsmem.NewIDGenerator())
	if err != nil {
		// notest
		return false, err
	}
	defer pLogEvent.Release()
	if err := as.Records().Apply(pLogEvent); err != nil {
		// notest
		return false, err
	}
	if err = as.Events().PutWlog(pLogEvent); err != nil {
		// notest
		return false, err
	}
	logger.Verbose("app workspace", as.AppQName(), appWSID.BaseWSID()-istructs.FirstBaseAppWSID, "(", appWSID, ") initialized")
	return true, nil
}
