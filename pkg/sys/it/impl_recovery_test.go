/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package sys_it

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	appdefsys "github.com/voedger/voedger/pkg/appdef/sys"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/checkpoints"
	"github.com/voedger/voedger/pkg/cluster"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/processors/actualizers"
	"github.com/voedger/voedger/pkg/sys/collection"
	"github.com/voedger/voedger/pkg/sys/workspace"
	it "github.com/voedger/voedger/pkg/vit"
	sys_test_template "github.com/voedger/voedger/pkg/vit/testdata"
	"github.com/voedger/voedger/pkg/vvm"
	builtinapps "github.com/voedger/voedger/pkg/vvm/builtin"
	vvmstorage "github.com/voedger/voedger/pkg/vvm/storage"
)

const (
	recoveryTestLoginName     = "recovery-test-login"
	recoveryTestLoginPassword = "recovery-test-password"
	recoveryTestWorkspaceName = "recovery-test-workspace"
	docWithNested             = `{"cuds": [
		{"fields":{"sys.ID": 1,"sys.QName": "app1pkg.Root", "FldRoot": 2}},
		{"fields":{"sys.ID": 2,"sys.QName": "app1pkg.Nested", "sys.ParentID":1,"sys.Container": "Nested","FldNested":3}},
		{"fields":{"sys.ID": 3,"sys.QName": "app1pkg.Third", "Fld1": 42,"sys.ParentID":2,"sys.Container": "Third"}}
	]}`
	singleDoc = `{"cuds":[
		{"fields":{"sys.ID":1,"sys.QName":"app1pkg.Root","FldRoot":3}}
	]}`
	singletonCUD = `{"cuds":[
		{"fields":{"sys.ID":1,"sys.QName":"app1pkg.Config","Fld1":"singleton tail"}}
	]}`
	nonIDBearingCommand = "c.sys.RegisterTempBLOB1d"
	cudBody1Doc         = `{"cuds":[
		{"fields":{"sys.ID":1,"sys.QName":"app1pkg.options"}}
	]}`
	fourAppWorkspaceDocs = `{"cuds":[
		{"fields":{"sys.ID":1,"sys.QName":"app1pkg.options"}},
		{"fields":{"sys.ID":2,"sys.QName":"app1pkg.options"}},
		{"fields":{"sys.ID":3,"sys.QName":"app1pkg.options"}},
		{"fields":{"sys.ID":4,"sys.QName":"app1pkg.options"}}
	]}`
)

type expectedWorkspaceNumbers struct {
	wsid                istructs.WSID
	wlogIDBearingOffset istructs.Offset
	nextWLogOffset      istructs.Offset
	nextRecordID        istructs.RecordID
}

func TestWorkspaceSequencesAfterRecovery(t *testing.T) {
	appQName := istructs.AppQName_test1_app1
	cfg := it.NewOwnVITConfig(
		it.WithApp(appQName, provideTestApp,
			it.WithWorkspaceTemplate(it.QNameApp1_TestWSKind, "test_template", sys_test_template.TestTemplateFS),
			it.WithUserLogin("login", "pwd"),
			it.WithChildWorkspace(it.QNameApp1_TestWSKind, "test_ws_1", "test_template", "", "login", map[string]interface{}{"IntFld": 42}),
			it.WithChildWorkspace(it.QNameApp1_TestWSKind, "test_ws_2", "test_template", "", "login", map[string]interface{}{"IntFld": 42}),
		),
	)

	var expectedWS1Numbers, expectedWS2Numbers expectedWorkspaceNumbers
	appID := istructs.ClusterApps[appQName]

	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()
	{
		require := require.New(t)
		workspace1 := vit.WS(appQName, "test_ws_1")
		workspace2 := vit.WS(appQName, "test_ws_2")
		appStructs, err := vit.IAppStructsProvider.BuiltIn(appQName)
		require.NoError(err)

		partition1, err := vit.IAppPartitions.AppWorkspacePartitionID(appQName, workspace1.WSID)
		require.NoError(err)
		partition2, err := vit.IAppPartitions.AppWorkspacePartitionID(appQName, workspace2.WSID)
		require.NoError(err)
		require.Equal(partition1, partition2)

		ws1Response := vit.PostWS(workspace1, "c.sys.CUD", docWithNested)
		require.Len(ws1Response.NewIDs, 3)
		expectedWS1Numbers = expectedWorkspaceNumbers{
			wsid:                workspace1.WSID,
			wlogIDBearingOffset: ws1Response.CurrentWLogOffset,
			nextRecordID:        ws1Response.NewIDs["3"] + 1,
		}

		singletonResponse := vit.PostWS(workspace1, "c.sys.CUD", singletonCUD)
		require.Equal(ws1Response.CurrentWLogOffset+1, singletonResponse.CurrentWLogOffset)
		expectedWS1Numbers.nextWLogOffset = singletonResponse.CurrentWLogOffset + 1

		// This must remain the partition's final event before restart. Recovery of
		// workspace 1 must not adopt workspace 2's independently advanced sequences.
		ws2Response := vit.PostWS(workspace2, "c.sys.CUD", docWithNested)
		for range 2 {
			ws2Response = vit.PostWS(workspace2, "c.sys.CUD", docWithNested)
		}
		require.Len(ws2Response.NewIDs, 3)
		require.Greater(ws2Response.CurrentWLogOffset+1, expectedWS1Numbers.nextWLogOffset)
		require.Greater(ws2Response.NewIDs["3"]+1, expectedWS1Numbers.nextRecordID)
		expectedWS2Numbers = expectedWorkspaceNumbers{
			wsid:                workspace2.WSID,
			wlogIDBearingOffset: ws2Response.CurrentWLogOffset,
			nextWLogOffset:      ws2Response.CurrentWLogOffset + 1,
			nextRecordID:        ws2Response.NewIDs["3"] + 1,
		}

		waitForActualizerFlush(t, appStructs, partition1)
	}
	require.NoError(t, vit.RestartVVM(it.WithAfterStop(func(storage it.RestartStorage) {
		checkpointStorage := vvmstorage.NewRecoveryCheckpointStorage(
			storage.AppStorage(t, istructs.AppQName_sys_vvm))
		checkWorkspaceCheckpoint(t, checkpointStorage, appID, expectedWS1Numbers)
		checkWorkspaceCheckpoint(t, checkpointStorage, appID, expectedWS2Numbers)
	})))
	{
		workspace1 := vit.WS(appQName, "test_ws_1")
		workspace2 := vit.WS(appQName, "test_ws_2")
		require.Equal(t, expectedWS1Numbers.wsid, workspace1.WSID)
		require.Equal(t, expectedWS2Numbers.wsid, workspace2.WSID)

		// insert a doc and check NewID and CurrentWLogOffset from the response
		checkNextWorkspaceSequence(vit, workspace1, expectedWS1Numbers)
		checkNextWorkspaceSequence(vit, workspace2, expectedWS2Numbers)
	}
}

func TestODocArgumentRecordIDsAreRespectedAfterRestart(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	appQName := istructs.AppQName_test1_app1
	cfg := it.NewSharedVITConfig(it.WithApp(appQName, provideTestApp,
		it.WithUserLogin(recoveryTestLoginName, recoveryTestLoginPassword),
	))
	var currentVIT *it.VIT
	t.Cleanup(func() {
		if currentVIT != nil {
			require.NoError(t, currentVIT.Shutdown())
		}
	})
	for _, test := range []struct {
		name             string
		removeCheckpoint bool
	}{
		{name: "with workspace checkpoint"},
		{name: "without workspace checkpoint", removeCheckpoint: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			const body = `{"args":{"sys.ID":1,"orecord1":[{"sys.ID":2,"sys.ParentID":1,"orecord2":[{"sys.ID":3,"sys.ParentID":2}]}]},"unloggedArgs":{"sys.ID":4}}`
			var workspace *it.AppWorkspace
			var lastWLogOffset istructs.Offset

			vit := it.NewVIT(t, &cfg)
			defer vit.TearDown()
			{
				currentVIT = vit
				owner := vit.GetPrincipal(appQName, recoveryTestLoginName)
				workspace = vit.CreateWorkspace(it.SimpleWSParams("ODoc-only-history-"+vit.NextName()), owner)

				// insert ODocs, issue IDs
				response := vit.PostWS(workspace, "c.app1pkg.CmdODocOne", body)
				require.Equal(t, map[string]istructs.RecordID{
					"1": 200001, "2": 200002, "3": 200003,
				}, response.NewIDs)
				lastWLogOffset = response.CurrentWLogOffset

				appStructs, err := vit.IAppStructsProvider.BuiltIn(appQName)
				require.NoError(t, err)
				foundODocEvent := false
				err = appStructs.Events().ReadWLog(t.Context(), workspace.WSID, istructs.FirstOffset, istructs.ReadToTheEnd,
					func(offset istructs.Offset, event istructs.IWLogEvent) error {
						defer event.Release()
						// The history must contain no ordinary IDs allocated by CUDs.
						for row := range event.CUDs {
							require.False(t, row.IsNew() && row.ID() >= istructs.FirstUserRecordID)
						}
						if offset == lastWLogOffset {
							foundODocEvent = true
							require.Empty(t, slices.Collect(event.CUDs))
							argument := event.ArgumentObject()
							require.Equal(t, response.NewIDs["1"], argument.AsRecord().ID())
							children := slices.Collect(argument.Children("orecord1"))
							require.Len(t, children, 1)
							require.Equal(t, response.NewIDs["2"], children[0].AsRecord().ID())
							grandchildren := slices.Collect(children[0].Children("orecord2"))
							require.Len(t, grandchildren, 1)
							require.Equal(t, response.NewIDs["3"], grandchildren[0].AsRecord().ID())
						}
						return nil
					})
				require.NoError(t, err)
				require.True(t, foundODocEvent)
				partitionID, err := vit.IAppPartitions.AppWorkspacePartitionID(appQName, workspace.WSID)
				require.NoError(t, err)
				waitForActualizerFlush(t, appStructs, partitionID)
			}
			require.NoError(t, vit.RestartVVM(it.WithAfterStop(func(storage it.RestartStorage) {
				appID := istructs.ClusterApps[appQName]
				checkpointAppStorage := storage.AppStorage(t, istructs.AppQName_sys_vvm)
				checkpointStorage := vvmstorage.NewRecoveryCheckpointStorage(checkpointAppStorage)
				checkpoint, found, err := checkpointStorage.GetWorkspaceCheckpoint(appID, workspace.WSID)
				require.NoError(t, err)
				if assert.True(t, found, "ODoc argument IDs must create a workspace checkpoint") {
					assert.Equal(t, lastWLogOffset, checkpoint.WLogIDBearingOffset,
						"ODoc argument IDs must advance the workspace checkpoint")
				}
				if test.removeCheckpoint {
					// Force a full WLog scan to show that fixing classification alone is insufficient.
					deleteWorkspaceCheckpoint(t, checkpointStorage, checkpointAppStorage, appID, workspace.WSID)
				}
			})))
			{
				require.Same(t, currentVIT, vit, "restart keeps the shared VIT identity")
				currentVIT = vit
				// 3 IDs for ODocs were already issued on the first run
				// so here next 3 IDs must be issued
				response := vit.PostWS(workspace, "c.app1pkg.CmdODocOne", body)
				require.Equal(t, lastWLogOffset+1, response.CurrentWLogOffset)
				assert.Equal(t, map[string]istructs.RecordID{
					"1": 200004, "2": 200005, "3": 200006,
				}, response.NewIDs, "recovery must preserve IDs consumed by ODoc arguments and their descendants")
			}
		})
	}
}

// 1st event: c.sys.Init inserts caller-supplied ID 200002
// 2nd: c.sys.CUD allocates ID 200003
// 3rd: c.sys.Init inserts caller-supplied ID 200001
// expected next: 200004
func TestDescendingSyncedRecordIDsDoNotBreakSequences(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	for _, checkpointMode := range []string{"projector checkpoint", "rebuilt checkpoint", "full WLog scan", "checkpoint on c.sys.Init event"} {
		t.Run(checkpointMode, func(t *testing.T) {
			require := require.New(t)
			appQName := istructs.AppQName_test1_app1
			cfg := it.NewOwnVITConfig(it.WithApp(appQName, provideTestApp,
				it.WithUserLogin(recoveryTestLoginName, recoveryTestLoginPassword),
			))
			var workspace *it.AppWorkspace
			var partitionID istructs.PartitionID
			var lastWLogOffset istructs.Offset

			vit := it.NewVIT(t, &cfg)
			defer vit.TearDown()
			{
				owner := vit.GetPrincipal(appQName, recoveryTestLoginName)
				workspace = vit.CreateWorkspace(it.SimpleWSParams("descending caller-supplied IDs"), owner)
				var err error
				partitionID, err = vit.IAppPartitions.AppWorkspacePartitionID(appQName, workspace.WSID)
				require.NoError(err)

				vit.PostWSSys(workspace, "c.sys.Init", `{"cuds":[
					{"fields":{"sys.ID":200002,"sys.QName":"app1pkg.Root","FldRoot":20}}
				]}`)
				generated := vit.PostWS(workspace, "c.sys.CUD", singleDoc)
				require.Equal(istructs.RecordID(200003), generated.NewID())
				withSuppliedID := vit.PostWSSys(workspace, "c.sys.Init", `{"cuds":[
					{"fields":{"sys.ID":200001,"sys.QName":"app1pkg.Root","FldRoot":10}}
				]}`)
				lastWLogOffset = withSuppliedID.CurrentWLogOffset
				require.Equal(generated.CurrentWLogOffset+1, lastWLogOffset)

				appStructs, err := vit.IAppStructsProvider.BuiltIn(appQName)
				require.NoError(err)
				var ordinaryIDs []istructs.RecordID
				err = appStructs.Events().ReadWLog(t.Context(), workspace.WSID, istructs.FirstOffset, istructs.ReadToTheEnd,
					func(_ istructs.Offset, event istructs.IWLogEvent) error {
						defer event.Release()
						for row := range event.CUDs {
							if row.IsNew() && row.ID() >= istructs.FirstUserRecordID {
								ordinaryIDs = append(ordinaryIDs, row.ID())
							}
						}
						return nil
					})
				require.NoError(err)

				require.Equal([]istructs.RecordID{200002, 200003, 200001}, ordinaryIDs)
			}
			require.NoError(vit.RestartVVM(it.WithAfterStop(func(storage it.RestartStorage) {
				// force full plog read out on the next run
				appID := istructs.ClusterApps[appQName]
				checkpointAppStorage := storage.AppStorage(t, istructs.AppQName_sys_vvm)
				checkpointStorage := vvmstorage.NewRecoveryCheckpointStorage(checkpointAppStorage)
				switch checkpointMode {
				case "rebuilt checkpoint":
					deletePartitionCheckpoint(t, checkpointStorage, checkpointAppStorage, appID, partitionID)
					deleteWorkspaceCheckpoint(t, checkpointStorage, checkpointAppStorage, appID, workspace.WSID)
				case "full WLog scan":
					deleteWorkspaceCheckpoint(t, checkpointStorage, checkpointAppStorage, appID, workspace.WSID)
				case "checkpoint on c.sys.Init event":
					require.NoError(checkpointStorage.PutWorkspaceCheckpoint(appID, workspace.WSID,
						checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: lastWLogOffset}))
				}
			})))
			{
				response := vit.PostWS(workspace, "c.sys.CUD", singleDoc)
				require.Equal(lastWLogOffset+1, response.CurrentWLogOffset)
				require.Equal(istructs.RecordID(200004), response.NewID(),
					"a c.sys.Init insert with a lower caller-supplied ID must not lower the recovered allocator")
				appStructs, err := vit.IAppStructsProvider.BuiltIn(appQName)
				require.NoError(err)
				original, err := appStructs.Records().Get(workspace.WSID, true, 200002)
				require.NoError(err)
				require.Equal(int32(20), original.AsInt32("FldRoot"),
					"allocating a record after restart must not overwrite a record inserted by c.sys.Init")
			}
		})
	}
}

func TestRecoveryFromCheckpoints(t *testing.T) {
	require := require.New(t)
	if testing.Short() {
		t.Skip()
	}

	t.Run("recovery: scn: Normal initial partition recovery", func(t *testing.T) {
		// Given an application declares partition IDs 0 through 2
		// And initial recovery does not fail
		cfg := recoveryVITConfig()
		appQName := istructs.AppQName_test1_app1
		appID := istructs.ClusterApps[appQName]
		var partition0LastPLogOffset istructs.Offset
		// And each partition is populated with a workspace having this history:
		// | partition ID | workspace | last WLog offset | last allocated ordinary record ID |
		// | 0            | Orders    | 7                | 200003                            |
		// | 1            | Billing   | 14               | 200005                            |
		// | 2            | Inventory | 20               | 200009                            |
		histories := []struct {
			partitionID    istructs.PartitionID
			name           string
			lastWLogOffset istructs.Offset
			lastRecordID   istructs.RecordID
			nextWLogOffset istructs.Offset
			nextRecordID   istructs.RecordID
			workspace      *it.AppWorkspace
		}{
			{partitionID: 0, name: "Orders", lastWLogOffset: 7, lastRecordID: 200003, nextWLogOffset: 8, nextRecordID: 200004},
			{partitionID: 1, name: "Billing", lastWLogOffset: 14, lastRecordID: 200005, nextWLogOffset: 15, nextRecordID: 200006},
			{partitionID: 2, name: "Inventory", lastWLogOffset: 20, lastRecordID: 200009, nextWLogOffset: 21, nextRecordID: 200010},
		}

		vit := it.NewVIT(t, &cfg)
		defer vit.TearDown()
		{
			owner := vit.GetPrincipal(appQName, recoveryTestLoginName)
			for index := range histories {
				history := &histories[index]
				history.workspace = createWorkspacesInPartition(t, vit, owner, history.partitionID, 1)[0]
				for id := istructs.FirstUserRecordID; id <= history.lastRecordID; id++ {
					response := vit.PostWS(history.workspace, "c.sys.CUD", singleDoc)
					require.Equal(id, response.NewID(), history.name)
				}
				fillWLogWithNonIDBearingEvents(t, vit, history.workspace, history.lastWLogOffset)
			}
			appStructs, err := vit.IAppStructsProvider.BuiltIn(appQName)
			require.NoError(err)
			for _, history := range histories {
				waitForActualizerFlush(t, appStructs, history.partitionID)
				require.Equal(history.lastWLogOffset, getLastWLogOffset(t, vit, history.workspace), history.name)
			}
			partition0LastPLogOffset = getLastPLogOffset(t, vit, appQName, 0)
		}
		require.NoError(vit.RestartVVM(it.WithAfterStop(func(storage it.RestartStorage) {
			checkpointAppStorage := storage.AppStorage(t, istructs.AppQName_sys_vvm)
			checkpointStorage := vvmstorage.NewRecoveryCheckpointStorage(checkpointAppStorage)
			// And partition 0 has a stored partition checkpoint
			require.NoError(checkpointStorage.PutPartitionCheckpoint(appID, 0,
				checkpoints.PartitionCheckpoint{LastPLogOffset: partition0LastPLogOffset}))
			// And partitions 1 and 2 have no stored partition checkpoints
			for _, history := range histories[1:] {
				deletePartitionCheckpoint(t, checkpointStorage, checkpointAppStorage, appID, history.partitionID)
				deleteWorkspaceCheckpoint(t, checkpointStorage, checkpointAppStorage, appID, history.workspace.WSID)
			}
		})))
		{
			// And VAOperator has started the VVM
			require.NotNil(vit.VoedgerVM)
			for _, history := range histories {
				// When a client inserts one ordinary record into each workspace, retrying "503 Service Unavailable" responses
				response := vit.PostWS(history.workspace, "c.sys.CUD", singleDoc)
				// Then each command eventually returns "200 OK"
				require.Equal(http.StatusOK, response.HTTPResp.StatusCode, history.name)
				// And the successful command responses contain:
				// | workspace | CurrentWLogOffset | new ordinary record ID |
				// | Orders    | 8                 | 200004                 |
				// | Billing   | 15                | 200006                 |
				// | Inventory | 21                | 200010                 |
				require.Equal(history.nextWLogOffset, response.CurrentWLogOffset, history.name)
				require.Equal(map[string]istructs.RecordID{"1": history.nextRecordID}, response.NewIDs, history.name)
			}
		}
	})

	t.Run("recovery: scn: At least the first two command requests to a partition and workspace return 503 after initial recovery", func(t *testing.T) {
		cfg := recoveryVITConfig()
		appQName := istructs.AppQName_test1_app1
		appID := istructs.ClusterApps[appQName]
		var workspace *it.AppWorkspace
		var partitionID istructs.PartitionID

		vit := it.NewVIT(t, &cfg)
		defer vit.TearDown()
		{
			owner := vit.GetPrincipal(appQName, recoveryTestLoginName)
			workspace = createWorkspacesInPartition(t, vit, owner, 1, 1)[0]
			var err error
			partitionID, err = vit.IAppPartitions.AppWorkspacePartitionID(appQName, workspace.WSID)
			require.NoError(err)
			vit.PostWS(workspace, "c.sys.CUD", cudBody1Doc)
		}
		require.NoError(vit.RestartVVM(it.WithAfterStop(func(storage it.RestartStorage) {
			checkpointAppStorage := storage.AppStorage(t, istructs.AppQName_sys_vvm)
			checkpointStorage := vvmstorage.NewRecoveryCheckpointStorage(checkpointAppStorage)
			deletePartitionCheckpoint(t, checkpointStorage, checkpointAppStorage, appID, partitionID)
			deleteWorkspaceCheckpoint(t, checkpointStorage, checkpointAppStorage, appID, workspace.WSID)
			_, ok, err := checkpointStorage.GetPartitionCheckpoint(appID, partitionID)
			require.NoError(err)
			require.False(ok)
		})))
		{
			// Given the server has become ready
			checkpointAppStorage, err := vit.IAppStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
			require.NoError(err)
			checkpointStorage := vvmstorage.NewRecoveryCheckpointStorage(checkpointAppStorage)
			_, partitionCheckpointExists, err := checkpointStorage.GetPartitionCheckpoint(appID, partitionID)
			require.NoError(err)
			require.True(partitionCheckpointExists)
			_, workspaceCheckpointExists, err := checkpointStorage.GetWorkspaceCheckpoint(appID, workspace.WSID)
			require.NoError(err)
			require.True(workspaceCheckpointExists)
			require.NotNil(vit.VoedgerVM)

			// When VAOperator submits command requests to a workspace in partition 1 in this order:
			// | request | timing               | retry policy            |
			// | 1       | after readiness      | none                    |
			// | 2       | after response 1     | none                    |
			responses := make([]int, 0, 2)
			for range 2 {
				response := vit.PostWS(workspace, "c.sys.CUD", cudBody1Doc,
					httpu.WithNoRetryPolicy(), httpu.Expect503())
				responses = append(responses, response.HTTPResp.StatusCode)
			}
			// Then the responses are:
			// | request | status                  |
			// | 1       | 503 Service Unavailable |
			// | 2       | 503 Service Unavailable |
			require.Equal([]int{http.StatusServiceUnavailable, http.StatusServiceUnavailable}, responses)

			// When VAOperator submits command requests to a workspace in partition 1 in this order:
			// | request | timing               | retry policy            |
			// | 3       | after response 2     | 503 Service Unavailable |
			response := vit.PostWS(workspace, "c.sys.CUD", cudBody1Doc)
			// Then the responses are:
			// | request | status                  |
			// | 3       | 200 OK                  |
			require.Equal(http.StatusOK, response.HTTPResp.StatusCode)
		}
	})

	t.Run("recovery: scn: A query does not trigger partition recovery", func(t *testing.T) {
		cfg := recoveryVITConfig()
		vit := it.NewVIT(t, &cfg)
		defer vit.TearDown()
		owner := vit.GetPrincipal(istructs.AppQName_test1_app1, recoveryTestLoginName)
		var workspaces []*it.AppWorkspace
		for partitionID := range istructs.PartitionID(3) {
			workspaces = append(workspaces, createWorkspacesInPartition(t, vit, owner, partitionID, 2)...)
		}

		for _, ws := range workspaces {
			vit.PostWS(ws, "c.sys.CUD", cudBody1Doc)
		}
		appStructs, err := vit.IAppStructsProvider.BuiltIn(istructs.AppQName_test1_app1)
		require.NoError(err)
		for partitionID := range istructs.NumAppPartitions(3) {
			waitForActualizerFlush(t, appStructs, istructs.PartitionID(partitionID))
		}

		// Given the server has become ready
		require.NoError(vit.RestartVVM())
		logs := logger.StartCapture(t, logger.LogLevelVerbose)
		for _, ws := range workspaces {
			// When VAOperator submits the very first query request to an existing workspace in any deployed partition without automatic retries
			response := vit.PostWS(ws, "q.sys.SqlQuery",
				`{"args":{"Query":"select * from sys.CollectionView where PartKey = 1"},"elements":[{"fields":["Result"]}]}`,
				httpu.WithNoRetryPolicy())
			// Then response status is "200 OK"
			require.Equal(http.StatusOK, response.HTTPResp.StatusCode)
		}
		// And no partition or workspace recovery is started by the query
		logs.NotContains("stage=cp.partition_recovery.start", "stage=cp.workspace_recovery.start")
	})

	t.Run("recovery: scn: Initial recovery failures are logged while the server becomes ready: partition 1", func(t *testing.T) {
		// | failed_partitions      | checkpoint_state                    |
		// | partition 1            | present for partitions 0 and 2 only |
		errInjector := newPLogReadFailure()
		cfg := recoveryVITConfigWithStorageHooks(errInjector, true)
		failed := []istructs.PartitionID{1}
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		vit := it.NewVIT(t, &cfg)
		defer vit.TearDown()
		owner := vit.GetPrincipal(istructs.AppQName_test1_app1, recoveryTestLoginName)
		workspaces := make([]*it.AppWorkspace, 3)
		for partitionID := range workspaces {
			workspaces[partitionID] = createWorkspacesInPartition(t, vit, owner, istructs.PartitionID(partitionID), 1)[0]
			vit.PostWS(workspaces[partitionID], "c.sys.CUD", cudBody1Doc)
		}
		// When VAOperator starts the VVM
		require.NoError(vit.RestartVVM(it.WithAfterStop(func(storage it.RestartStorage) {
			// Given application "test1/app1" declares partition IDs 0 through 2 with no stored partition checkpoints
			removeAppPartitionCheckpoints(t, storage, istructs.NumAppPartitions(3))
			// And initial recovery fails for <failed_partitions> with "PLog read failed"
			// failed_partitions = partition 1
			errInjector.failPLogReads(failed)
			logCap.Reset()
		})))
		// Then partition checkpoints are <checkpoint_state>
		// checkpoint_state = present for partitions 0 and 2 only
		requirePartitionCheckpointPresence(t, vit, failed)
		// And initial-recovery error "PLog read failed" is logged for <failed_partitions>
		// failed_partitions = partition 1
		requireInitialRecoveryFailureLogs(t, logCap, failed)
		// And the server becomes ready
		require.NotNil(vit.VoedgerVM)
	})

	t.Run("recovery: scn: Initial recovery failures are logged while the server becomes ready: partitions 0 through 2", func(t *testing.T) {
		// | failed_partitions      | checkpoint_state                    |
		// | partitions 0 through 2 | absent for every partition          |
		errInjector := newPLogReadFailure()
		cfg := recoveryVITConfigWithStorageHooks(errInjector, true)
		failed := []istructs.PartitionID{0, 1, 2}
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		vit := it.NewVIT(t, &cfg)
		defer vit.TearDown()
		owner := vit.GetPrincipal(istructs.AppQName_test1_app1, recoveryTestLoginName)
		workspaces := make([]*it.AppWorkspace, 3)
		for partitionID := range workspaces {
			workspaces[partitionID] = createWorkspacesInPartition(t, vit, owner, istructs.PartitionID(partitionID), 1)[0]
			vit.PostWS(workspaces[partitionID], "c.sys.CUD", cudBody1Doc)
		}
		// When VAOperator starts the VVM
		require.NoError(vit.RestartVVM(it.WithAfterStop(func(storage it.RestartStorage) {
			// Given application "test1/app1" declares partition IDs 0 through 2 with no stored partition checkpoints
			removeAppPartitionCheckpoints(t, storage, istructs.NumAppPartitions(3))
			// And initial recovery fails for <failed_partitions> with "PLog read failed"
			// failed_partitions = partitions 0 through 2
			errInjector.failPLogReads(failed)
			logCap.Reset()
		})))
		// Then partition checkpoints are <checkpoint_state>
		// checkpoint_state = absent for every partition
		requirePartitionCheckpointPresence(t, vit, failed)
		// And initial-recovery error "PLog read failed" is logged for <failed_partitions>
		// failed_partitions = partitions 0 through 2
		requireInitialRecoveryFailureLogs(t, logCap, failed)
		// And the server becomes ready
		require.NotNil(vit.VoedgerVM)
	})

	t.Run("recovery: scn: Commands return 500 on a partition whose initial recovery failed", func(t *testing.T) {
		errInjector := newPLogReadFailure()
		cfg := recoveryVITConfigWithStorageHooks(errInjector, true)
		appQName := istructs.AppQName_test1_app1
		vit := it.NewVIT(t, &cfg)
		defer vit.TearDown()
		owner := vit.GetPrincipal(appQName, recoveryTestLoginName)
		// And workspace Billing belongs to partition 1
		billing := createWorkspacesInPartition(t, vit, owner, 1, 1)[0]
		// And VAOperator has started the VVM
		require.NoError(vit.RestartVVM(it.WithAfterStop(func(storage it.RestartStorage) {
			// Given application "test1/app1" declares partition IDs 0 through 2 with no stored partition checkpoints
			removeAppPartitionCheckpoints(t, storage, istructs.NumAppPartitions(3))
			// And initial recovery fails for partition 1 with "PLog read failed"
			errInjector.failPLogReads([]istructs.PartitionID{1})
		})))
		// When a client inserts one ordinary record into Billing without retries
		response := vit.PostWS(billing, "c.sys.CUD", singleDoc, httpu.WithNoRetryPolicy(), httpu.Expect500())
		// Then the command returns "500 Internal Server Error"
		require.Equal(http.StatusInternalServerError, response.HTTPResp.StatusCode)
		// And its response error is "partition not found"
		requirePartitionNotFound(t, response.SysError)
	})

	t.Run("recovery: scn: The router receives the full partition count from the application descriptor despite initial recovery failures: partition 1", func(t *testing.T) {
		// | failed_partitions      |
		// | partition 1            |
		errInjector := newPLogReadFailure()
		writes := newAppLogWriteRecorder()
		cfg := recoveryVITConfigWithStorageHooks(errInjector, false, recoveryStorageObservers{appLogWrites: writes})
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		var history recordIDBeforeRestart
		vit := it.NewVIT(t, &cfg)
		defer vit.TearDown()
		{
			// Given application "test1/app1" has a deployment descriptor with NumPartitions 3
			requireDeploymentDescriptorCount(t, vit.IAppStructsProvider, 3)
			// And workspace 10 has existing data in partition 1
			history = createRecord(t, vit)
			writes.requireLogWrites(t)
		}
		require.NoError(vit.RestartVVM(it.WithAfterStop(func(storage it.RestartStorage) {
			removeAppPartitionCheckpoints(t, storage, 3)
			// And initial recovery fails for <failed_partitions>
			// failed_partitions = partition 1
			errInjector.failPLogReads([]istructs.PartitionID{1})
		})))
		{
			// And VAOperator has started the VVM
			require.NotNil(vit.VoedgerVM)
			require.ElementsMatch([]istructs.PartitionID{0, 2}, deployedPartitionIDs(t, vit))
			checkCorrectWSAfterCheckpointFailure(t, vit, history, writes, logCap)
		}
	})

	t.Run("recovery: scn: The router receives the full partition count from the application descriptor despite initial recovery failures: all partitions", func(t *testing.T) {
		// | failed_partitions      |
		// | partitions 0 through 2 |
		errInjector := newPLogReadFailure()
		writes := newAppLogWriteRecorder()
		cfg := recoveryVITConfigWithStorageHooks(errInjector, false, recoveryStorageObservers{appLogWrites: writes})
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		var history recordIDBeforeRestart
		vit := it.NewVIT(t, &cfg)
		defer vit.TearDown()
		{
			// Given application "test1/app1" has a deployment descriptor with NumPartitions 3
			requireDeploymentDescriptorCount(t, vit.IAppStructsProvider, 3)
			// And workspace 10 has existing data in partition 1
			history = createRecord(t, vit)
			writes.requireLogWrites(t)
		}
		require.NoError(vit.RestartVVM(it.WithAfterStop(func(storage it.RestartStorage) {
			removeAppPartitionCheckpoints(t, storage, 3)
			// And initial recovery fails for <failed_partitions>
			// failed_partitions = partitions 0 through 2
			errInjector.failPLogReads([]istructs.PartitionID{0, 1, 2})
		})))
		{
			// And VAOperator has started the VVM
			require.NotNil(vit.VoedgerVM)
			require.Empty(deployedPartitionIDs(t, vit))
			checkCorrectWSAfterCheckpointFailure(t, vit, history, writes, logCap)
		}
	})

	t.Run("recovery: scn: A changed app descriptor partition count causes panic", func(t *testing.T) {
		errInjector := newPLogReadFailure()
		writes := newAppLogWriteRecorder()
		cfg := recoveryVITConfigWithStorageHooks(errInjector, false, recoveryStorageObservers{appLogWrites: writes})
		vit := it.NewVIT(t, &cfg)
		defer vit.TearDown()
		// Given application "test1/app1" has a persisted deployment descriptor with NumPartitions 3
		requireDeploymentDescriptorCount(t, vit.IAppStructsProvider, 3)
		// And all declared partitions are deployed
		require.ElementsMatch([]istructs.PartitionID{0, 1, 2}, deployedPartitionIDs(t, vit))
		requireChangedPartitionCountRestartRejected(t, vit, writes)
	})

	t.Run("recovery: scn: VVM launch fails with an error on failure of the initial recovery of a partition of the Cluster app", func(t *testing.T) {
		errInjector := newPLogReadFailure()
		// Given initial recovery fails for the cluster application's sole partition
		errInjector.failPLogReads([]istructs.PartitionID{0})
		cfg := vvm.GetTestVVMCfg(httpu.LocalhostIP)
		cfg.AdminPort = 0
		underlyingFactory := cfg.StorageFactory
		cfg.StorageFactory = func(currentTime timeu.ITime) (istorage.IAppStorageFactory, error) {
			underlying, err := underlyingFactory(currentTime)
			if err != nil {
				return nil, err
			}
			return &readErrorInjectingAndWriteRecordingStorageFactory{
				IAppStorageFactory: underlying,
				errInjector:        errInjector,
				appStoragePrefix:   "syscluster",
			}, nil
		}
		voedgerVM, err := vvm.Provide(cfg)
		require.NoError(err)
		t.Cleanup(func() {
			err := voedgerVM.Shutdown()
			require.ErrorContains(err, "failed to initialize or deploy cluster app partition")
			require.ErrorIs(err, vvm.ErrVVMServicesLaunch)
		})
		logs := logger.StartCapture(t, logger.LogLevelVerbose)
		// When VAOperator starts the VVM
		problemCtx := voedgerVM.Launch(vvm.DefaultLeadershipDurationSeconds, vvm.DefaultLeadershipAcquisitionDuration)
		// Then VVM launch fails with an error
		require.Error(problemCtx.Err())
		logs.HasLine("stage=bootstrap.initial_recovery.error", "app=sys/cluster", "partid=0", "PLog read failed")
		_, err = voedgerVM.IAppPartitions.AppPartsCount(istructs.AppQName_test1_app1)
		require.ErrorIs(err, appparts.ErrNotFound)
	})

	t.Run("recovery: scn: A later bootstrap deploys a previously failed partition after successful initial recovery", func(t *testing.T) {
		errInjector := newPLogReadFailure()
		cfg := recoveryVITConfigWithStorageHooks(errInjector, true)
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		vit := it.NewVIT(t, &cfg)
		defer vit.TearDown()
		// And workspace Billing in partition 1 has last WLog offset 14 and last allocated ordinary record ID 200005
		owner := vit.GetPrincipal(istructs.AppQName_test1_app1, recoveryTestLoginName)
		billing := createWorkspacesInPartition(t, vit, owner, 1, 1)[0]
		for id := istructs.FirstUserRecordID; id <= 200005; id++ {
			response := vit.PostWS(billing, "c.sys.CUD", singleDoc)
			require.Equal(id, response.NewID())
		}
		fillWLogWithNonIDBearingEvents(t, vit, billing, 14)
		appStructs, err := vit.IAppStructsProvider.BuiltIn(istructs.AppQName_test1_app1)
		require.NoError(err)
		waitForActualizerFlush(t, appStructs, 1)
		require.Equal(istructs.Offset(14), getLastWLogOffset(t, vit, billing))
		// Given initial recovery of partition 1 of application "test1/app1" failed on the previous VVM start
		require.NoError(vit.RestartVVM(it.WithAfterStop(func(storage it.RestartStorage) {
			removeAppPartitionCheckpoints(t, storage, 3)
			errInjector.failPLogReads([]istructs.PartitionID{1})
			logCap.Reset()
		})))
		logCap.HasLine("stage=bootstrap.initial_recovery.error", "app=test1/app1", "partid=1", "PLog read failed")
		// When VAOperator starts the VVM again and initial recovery of partition 1 succeeds
		require.NoError(vit.RestartVVM(it.WithAfterStop(func(_ it.RestartStorage) {
			errInjector.resolve()
		})))
		// Then inserting an ordinary record into Billing, retrying "503 Service Unavailable" responses, returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006
		response := vit.PostWS(billing, "c.sys.CUD", singleDoc)
		require.Equal(http.StatusOK, response.HTTPResp.StatusCode)
		require.Equal(istructs.Offset(15), response.CurrentWLogOffset)
		require.Equal(map[string]istructs.RecordID{"1": 200006}, response.NewIDs)
	})

	t.Run("recovery: scn: Invalid partition checkpoint causes PLog read out from beginning: checkpoint 1000000", func(t *testing.T) {
		// | plog            | lastPLogOffset | lastWLogOffset | lastRecordID |
		// | 1..50           | 1000000        | 14             | 200005       |
		// And workspace "recovery-test-workspace" has last WLog offset <lastWLogOffset> and record ID allocation baseline <lastRecordID>
		// lastWLogOffset = 14
		// lastRecordID = 200005
		f := newRecoveryCheckpointFixture(t, checkpointTypePartition, checkpointBeyondTail)

		//                                    PLog end                      invalid checkpoint
		//                                        |                                  |
		//                                        v                                  v
		// PLog: -- 1 -- 2 -- ... -- 48 -- 49 -- 50       ... no events ...       1000000
		//          \___ fallback recovery scan __/                                  |
		//          ^                                                      stored lastPLogOffset
		//     FirstOffset
		// Initial scan at 1000000 returns no events, so recovery scans offsets 1 through 50.
		// The next PLog offset is 51.

		// Given the PLog event offsets are <plog>
		// plog = 1..50
		// And the stored partition checkpoint is <lastPLogOffset>
		// lastPLogOffset = 1000000
		// When VAOperator restarts the VVM
		f.restart()

		// Then inserting an ordinary record into "recovery-test-workspace", retrying "503 Service Unavailable" responses, returns "200 OK" with CurrentWLogOffset <lastWLogOffset> + 1 and new ordinary record ID <lastRecordID> + 1
		// lastWLogOffset = 14
		// lastRecordID = 200005
		response := f.vit.PostWS(f.workspace, "c.sys.CUD", singleDoc)
		require.Equal(http.StatusOK, response.HTTPResp.StatusCode)
		require.Equal(istructs.Offset(15), response.CurrentWLogOffset)
		require.Equal(map[string]istructs.RecordID{"1": 200006}, response.NewIDs)
		f.requirePLogTail(response.CurrentWLogOffset)
		f.requireCheckpointLogs()
	})

	t.Run("recovery: scn: Invalid partition checkpoint causes PLog read out from beginning: checkpoint 1", func(t *testing.T) {
		// | plog            | lastPLogOffset | lastWLogOffset | lastRecordID |
		// | 2..50           | 1              | 14             | 200005       |
		// And workspace "recovery-test-workspace" has last WLog offset <lastWLogOffset> and record ID allocation baseline <lastRecordID>
		// lastWLogOffset = 14
		// lastRecordID = 200005
		f := newRecoveryCheckpointFixture(t, checkpointTypePartition, checkpointBeforeBeginning)

		// Given the PLog event offsets are <plog>
		// plog = 2..50
		// And the stored partition checkpoint is <lastPLogOffset>
		// lastPLogOffset = 1
		// When VAOperator restarts the VVM
		f.restart()

		// Then inserting an ordinary record into "recovery-test-workspace", retrying "503 Service Unavailable" responses, returns "200 OK" with CurrentWLogOffset <lastWLogOffset> + 1 and new ordinary record ID <lastRecordID> + 1
		// lastWLogOffset = 14
		// lastRecordID = 200005
		response := f.vit.PostWS(f.workspace, "c.sys.CUD", singleDoc)
		require.Equal(http.StatusOK, response.HTTPResp.StatusCode)
		require.Equal(istructs.Offset(15), response.CurrentWLogOffset)
		require.Equal(map[string]istructs.RecordID{"1": 200006}, response.NewIDs)
		f.requirePLogTail(response.CurrentWLogOffset)
		f.requireCheckpointLogs()
	})

	t.Run("recovery: scn: Invalid partition checkpoint causes PLog read out from beginning: checkpoint 10", func(t *testing.T) {
		// | plog            | lastPLogOffset | lastWLogOffset | lastRecordID |
		// | 1..50 except 10 | 10             | 14             | 200005       |
		// And workspace "recovery-test-workspace" has last WLog offset <lastWLogOffset> and record ID allocation baseline <lastRecordID>
		// lastWLogOffset = 14
		// lastRecordID = 200005
		f := newRecoveryCheckpointFixture(t, checkpointTypePartition, checkpointMissingEvent)

		//                               stored lastPLogOffset                       PLog end
		//                                         |                                     |
		//                                         v                                     v
		// PLog: -- 1 -- 2 -- ... -- 8 -- 9 -- [10 missing] -- 11 -- 12 -- ... -- 49 -- 50
		//          \______________________ fallback recovery scan ______________________/
		//          ^                                          ^
		//     FirstOffset                              first callback
		// Initial read from checkpoint 10 returns event 11 first, so validation fails.
		// Recovery rescans from FirstOffset through offset 50; the next PLog offset is 51.

		// Given the PLog event offsets are <plog>
		// plog = 1..50 except 10
		// And the stored partition checkpoint is <lastPLogOffset>
		// lastPLogOffset = 10
		// When VAOperator restarts the VVM
		f.restart()

		// Then inserting an ordinary record into "recovery-test-workspace", retrying "503 Service Unavailable" responses, returns "200 OK" with CurrentWLogOffset <lastWLogOffset> + 1 and new ordinary record ID <lastRecordID> + 1
		// lastWLogOffset = 14
		// lastRecordID = 200005
		response := f.vit.PostWS(f.workspace, "c.sys.CUD", singleDoc)
		require.Equal(http.StatusOK, response.HTTPResp.StatusCode)
		require.Equal(istructs.Offset(15), response.CurrentWLogOffset)
		require.Equal(map[string]istructs.RecordID{"1": 200006}, response.NewIDs)
		f.requirePLogTail(response.CurrentWLogOffset)
		f.requireCheckpointLogs()
	})

	t.Run("recovery: scn: Workspace recovery preserves sequence numbers after a mixed non-ID-bearing WLog tail", func(t *testing.T) {
		// Given the WLog has an ID-bearing event at offset 6 with record ID 200001
		// And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
		// And the WLog has non-CUD events at offsets 11 and 12
		// And the WLog has a singleton insert at offset 13
		// And the WLog has an update of record 200002 at offset 14
		// And the WLog ends at offset 14
		f := newRecoveryCheckpointFixture(t, checkpointTypeWorkspace, checkpointMixedTail)

		// And the stored workspace checkpoint is wlogIDBearingOffset 10
		// And the VVM has restarted and completed partition recovery
		f.restart()
		f.recoverPartition()

		// When VAOperator submits an ordinary record insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
		response := f.vit.PostWS(f.workspace, "c.sys.CUD", singleDoc)
		// Then the insert returns "200 OK" with CurrentWLogOffset 15 and NewIDs {"1": 200006}
		require.Equal(http.StatusOK, response.HTTPResp.StatusCode)
		require.Equal(istructs.Offset(15), response.CurrentWLogOffset)
		require.Equal(map[string]istructs.RecordID{"1": 200006}, response.NewIDs)
		f.requirePLogTail(response.CurrentWLogOffset)
		f.requireCheckpointLogs()
	})

	t.Run("recovery: scn: Workspace without user record IDs starts allocation at FirstUserRecordID", func(t *testing.T) {
		// Given the WLog has three non-ID-bearing events at offsets 1 through 3
		f := newRecoveryCheckpointFixture(t, checkpointTypeWorkspace, checkpointNoUserIDs)

		// And the stored workspace checkpoint is wlogIDBearingOffset FirstOffset
		// And the VVM has restarted and completed partition recovery
		f.restart()
		f.recoverPartition()

		// When VAOperator submits an ordinary record insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
		response := f.vit.PostWS(f.workspace, "c.sys.CUD", singleDoc)
		// Then the insert returns "200 OK" with CurrentWLogOffset 4 and new ordinary record ID FirstUserRecordID (200001)
		require.Equal(http.StatusOK, response.HTTPResp.StatusCode)
		require.Equal(istructs.Offset(4), response.CurrentWLogOffset)
		require.Equal(map[string]istructs.RecordID{"1": 200001}, response.NewIDs)
		f.requirePLogTail(response.CurrentWLogOffset)
		f.requireCheckpointLogs()
	})

	t.Run("recovery: scn: Invalid workspace checkpoint falls back to the WLog beginning: wlogIDBearingOffset 15 beyond the WLog end", func(t *testing.T) {
		// | checkpoint                                                   |
		// | wlogIDBearingOffset 15 beyond the WLog end                   |
		// Given the WLog has an ID-bearing event at offset 6 with record ID 200001
		// And the WLog has a non-ID-bearing event at offset 9
		// And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
		// And the WLog has a CUD-free event at offset 12
		// And the WLog has a singleton-only event at offset 13
		// And the WLog has an update of an existing record at offset 14
		// And the WLog ends at offset 14
		f := newRecoveryCheckpointFixture(t, checkpointTypeWorkspace, checkpointBeyondTail)

		//                                                            WLog end    invalid checkpoint
		//                                                                |             |
		//                                                                v             v
		// WLog: -- 1 -- 2 -- ... -- 6 -- ... -- 10 -- 11 -- 12 -- 13 -- 14           [15 absent]
		//          \_______________ fallback recovery scan ______________/             |
		//          ^                                                             wlogIDBearingOffset 15
		//     FirstOffset
		// Initial read at checkpoint 15 returns no events, so recovery scans offsets 1 through 14.
		// The latest ID-bearing event is at offset 10; later events allocate no ordinary record IDs.
		// The next insert uses WLog offset 15 and record ID 200006.

		// And the stored workspace checkpoint is <checkpoint>
		// checkpoint = wlogIDBearingOffset 15 beyond the WLog end
		// And the VVM has restarted and completed partition recovery
		f.restart()
		f.recoverPartition()

		// When VAOperator submits an insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
		response := f.vit.PostWS(f.workspace, "c.sys.CUD", singleDoc)
		// Then the insert returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006
		require.Equal(http.StatusOK, response.HTTPResp.StatusCode)
		require.Equal(istructs.Offset(15), response.CurrentWLogOffset)
		require.Equal(map[string]istructs.RecordID{"1": 200006}, response.NewIDs)
		f.requirePLogTail(response.CurrentWLogOffset)
		// And an ERROR log at stage "cp.workspace_recovery.checkpoint.invalid" identifies the stored WLog offset
		// And its message contains "does not identify an ID-bearing event; retrying from FirstOffset"
		f.requireCheckpointLogs()
	})

	t.Run("recovery: scn: Invalid workspace checkpoint falls back to the WLog beginning: wlogIDBearingOffset 8 whose first returned event is 9", func(t *testing.T) {
		// | checkpoint                                                   |
		// | wlogIDBearingOffset 8 whose first returned event is 9        |
		// Given the WLog has an ID-bearing event at offset 6 with record ID 200001
		// And the WLog has a non-ID-bearing event at offset 9
		// And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
		// And the WLog has a CUD-free event at offset 12
		// And the WLog has a singleton-only event at offset 13
		// And the WLog has an update of an existing record at offset 14
		// And the WLog ends at offset 14
		f := newRecoveryCheckpointFixture(t, checkpointTypeWorkspace, checkpointMissingEvent)

		//                                stored checkpoint                             WLog end
		//                                        |                                         |
		//                                        v                                         v
		// WLog: -- 1 -- 2 -- ... -- 6 -- 7 -- [8 missing] -- 9 -- 10 -- 11 -- 12 -- 13 -- 14
		//          \________________________ fallback recovery scan _______________________/
		//          ^                                         ^
		//     FirstOffset                             first callback
		// Initial read from wlogIDBearingOffset 8 returns event 9 first, so validation fails.
		// Recovery rescans from FirstOffset through offset 14, including the ID-bearing event at 10.
		// The next insert uses WLog offset 15 and record ID 200006.

		// And the stored workspace checkpoint is <checkpoint>
		// checkpoint = wlogIDBearingOffset 8 whose first returned event is 9
		// And the VVM has restarted and completed partition recovery
		f.restart()
		f.recoverPartition()

		// When VAOperator submits an insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
		response := f.vit.PostWS(f.workspace, "c.sys.CUD", singleDoc)
		// Then the insert returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006
		require.Equal(http.StatusOK, response.HTTPResp.StatusCode)
		require.Equal(istructs.Offset(15), response.CurrentWLogOffset)
		require.Equal(map[string]istructs.RecordID{"1": 200006}, response.NewIDs)
		f.requirePLogTail(response.CurrentWLogOffset)
		// And an ERROR log at stage "cp.workspace_recovery.checkpoint.invalid" identifies the stored WLog offset
		// And its message contains "does not identify an ID-bearing event; retrying from FirstOffset"
		f.requireCheckpointLogs()
	})

	t.Run("recovery: scn: Invalid workspace checkpoint falls back to the WLog beginning: wlogIDBearingOffset 12 pointing to a CUD-free event", func(t *testing.T) {
		// | checkpoint                                                   |
		// | wlogIDBearingOffset 12 pointing to a CUD-free event          |
		// Given the WLog has an ID-bearing event at offset 6 with record ID 200001
		// And the WLog has a non-ID-bearing event at offset 9
		// And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
		// And the WLog has a CUD-free event at offset 12
		// And the WLog has a singleton-only event at offset 13
		// And the WLog has an update of an existing record at offset 14
		// And the WLog ends at offset 14
		f := newRecoveryCheckpointFixture(t, checkpointTypeWorkspace, checkpointCUDFree)

		//                                          CUD-free checkpoint
		//                                                            WLog end
		//                                                   |            |
		//                                                   v            v
		// WLog: -- 1 -- 2 -- ... -- 6 -- ... -- 10 -- 11 -- 12 -- 13 -- 14
		//          \_______________ fallback recovery scan ______________/
		//     FirstOffset
		// Event 12 exists but allocates no ordinary record IDs, so wlogIDBearingOffset 12 is invalid.
		// Recovery rescans from FirstOffset and finds the latest ID-bearing event at offset 10.
		// The next insert uses WLog offset 15 and record ID 200006.

		// And the stored workspace checkpoint is <checkpoint>
		// checkpoint = wlogIDBearingOffset 12 pointing to a CUD-free event
		// And the VVM has restarted and completed partition recovery
		f.restart()
		f.recoverPartition()

		// When VAOperator submits an insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
		response := f.vit.PostWS(f.workspace, "c.sys.CUD", singleDoc)
		// Then the insert returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006
		require.Equal(http.StatusOK, response.HTTPResp.StatusCode)
		require.Equal(istructs.Offset(15), response.CurrentWLogOffset)
		require.Equal(map[string]istructs.RecordID{"1": 200006}, response.NewIDs)
		f.requirePLogTail(response.CurrentWLogOffset)
		// And an ERROR log at stage "cp.workspace_recovery.checkpoint.invalid" identifies the stored WLog offset
		// And its message contains "does not identify an ID-bearing event; retrying from FirstOffset"
		f.requireCheckpointLogs()
	})

	t.Run("recovery: scn: Invalid workspace checkpoint falls back to the WLog beginning: wlogIDBearingOffset 13 pointing to a singleton-only event", func(t *testing.T) {
		// | checkpoint                                                   |
		// | wlogIDBearingOffset 13 pointing to a singleton-only event    |
		// Given the WLog has an ID-bearing event at offset 6 with record ID 200001
		// And the WLog has a non-ID-bearing event at offset 9
		// And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
		// And the WLog has a CUD-free event at offset 12
		// And the WLog has a singleton-only event at offset 13
		// And the WLog has an update of an existing record at offset 14
		// And the WLog ends at offset 14
		f := newRecoveryCheckpointFixture(t, checkpointTypeWorkspace, checkpointSingleton)

		// And the stored workspace checkpoint is <checkpoint>
		// checkpoint = wlogIDBearingOffset 13 pointing to a singleton-only event
		// And the VVM has restarted and completed partition recovery
		f.restart()
		f.recoverPartition()

		// When VAOperator submits an insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
		response := f.vit.PostWS(f.workspace, "c.sys.CUD", singleDoc)
		// Then the insert returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006
		require.Equal(http.StatusOK, response.HTTPResp.StatusCode)
		require.Equal(istructs.Offset(15), response.CurrentWLogOffset)
		require.Equal(map[string]istructs.RecordID{"1": 200006}, response.NewIDs)
		f.requirePLogTail(response.CurrentWLogOffset)
		// And an ERROR log at stage "cp.workspace_recovery.checkpoint.invalid" identifies the stored WLog offset
		// And its message contains "does not identify an ID-bearing event; retrying from FirstOffset"
		f.requireCheckpointLogs()
	})

	t.Run("recovery: scn: Invalid workspace checkpoint falls back to the WLog beginning: wlogIDBearingOffset 14 pointing to an existing-record update", func(t *testing.T) {
		// | checkpoint                                                   |
		// | wlogIDBearingOffset 14 pointing to an existing-record update |
		// Given the WLog has an ID-bearing event at offset 6 with record ID 200001
		// And the WLog has a non-ID-bearing event at offset 9
		// And the WLog has an ID-bearing event at offset 10 with record IDs 200002 through 200005
		// And the WLog has a CUD-free event at offset 12
		// And the WLog has a singleton-only event at offset 13
		// And the WLog has an update of an existing record at offset 14
		// And the WLog ends at offset 14
		f := newRecoveryCheckpointFixture(t, checkpointTypeWorkspace, checkpointExistingRecordUpdate)

		// And the stored workspace checkpoint is <checkpoint>
		// checkpoint = wlogIDBearingOffset 14 pointing to an existing-record update
		// And the VVM has restarted and completed partition recovery
		f.restart()
		f.recoverPartition()

		// When VAOperator submits an insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
		response := f.vit.PostWS(f.workspace, "c.sys.CUD", singleDoc)
		// Then the insert returns "200 OK" with CurrentWLogOffset 15 and new ordinary record ID 200006
		require.Equal(http.StatusOK, response.HTTPResp.StatusCode)
		require.Equal(istructs.Offset(15), response.CurrentWLogOffset)
		require.Equal(map[string]istructs.RecordID{"1": 200006}, response.NewIDs)
		f.requirePLogTail(response.CurrentWLogOffset)
		// And an ERROR log at stage "cp.workspace_recovery.checkpoint.invalid" identifies the stored WLog offset
		// And its message contains "does not identify an ID-bearing event; retrying from FirstOffset"
		f.requireCheckpointLogs()
	})

}

type checkpointType int

const (
	checkpointTypePartition checkpointType = iota
	checkpointTypeWorkspace
)

type checkpointTestCase int

const (
	checkpointAbsent checkpointTestCase = iota
	checkpointZero
	checkpointValidButOutdated
	checkpointValid
	checkpointAtWLogTail
	checkpointBeyondTail
	checkpointMissingEvent
	checkpointCUDFree
	checkpointSingleton
	checkpointExistingRecordUpdate
	checkpointMixedTail
	checkpointNoUserIDs
	checkpointBeforeBeginning
)

const (
	testOffsetPLogTail        istructs.Offset = 50
	testOffsetBeyondPLogTail  istructs.Offset = 1_000_000
	testOffsetInitialWLogTail istructs.Offset = 3
	testOffsetFirstIDBearing  istructs.Offset = 6
	testOffsetWLogIDBearing   istructs.Offset = 10
	testOffsetCUDFree         istructs.Offset = 12
	testOffsetSingleton       istructs.Offset = 13
	testOffsetWLogTail        istructs.Offset = 14
	testOffsetMissingWLog     istructs.Offset = 8
)

type recoveryCheckpointFixture struct {
	t           *testing.T
	vit         *it.VIT
	workspace   *it.AppWorkspace
	appStructs  istructs.IAppStructs
	partitionID istructs.PartitionID
	expected    expectedWorkspaceNumbers
	logs        logger.ILogCaptor
	scope       checkpointType
	kind        checkpointTestCase
}

func newRecoveryCheckpointFixture(t *testing.T, scope checkpointType, kind checkpointTestCase) *recoveryCheckpointFixture {
	t.Helper()
	require := require.New(t)
	appQName := istructs.AppQName_test1_app1
	cfg := it.NewOwnVITConfig(it.WithApp(appQName, provideTestApp,
		it.WithUserLogin(recoveryTestLoginName, recoveryTestLoginPassword),
		it.WithChildWorkspace(it.QNameApp1_TestWSKind, recoveryTestWorkspaceName, "", "", recoveryTestLoginName, map[string]interface{}{"IntFld": 42})))
	logCap := logger.StartCapture(t, logger.LogLevelVerbose)

	var (
		partitionID istructs.PartitionID
		expected    expectedWorkspaceNumbers
		currentWS   *it.AppWorkspace
		appStructs  istructs.IAppStructs
	)
	vit := it.NewVIT(t, &cfg)
	t.Cleanup(vit.TearDown)
	{
		currentWS = vit.WS(appQName, recoveryTestWorkspaceName)
		var err error
		partitionID, err = vit.IAppPartitions.AppWorkspacePartitionID(appQName, currentWS.WSID)
		require.NoError(err)
		appStructs, err = vit.IAppStructsProvider.BuiltIn(appQName)
		require.NoError(err)
		plogOffsets := vit.SubscribeForN10nProjectionKey(in10n.ProjectionKey{
			App:        appQName,
			Projection: actualizers.PLogUpdatesQName,
			WS:         istructs.WSID(partitionID),
		})
		collectionOffsets := vit.SubscribeForN10n(currentWS, collection.QNameCollectionView)

		//                          ID-bearing                ID-bearing
		//                          one new ID              four new IDs             WLog end
		//                              |                         |                      |
		//                              v                         v                      v
		// WLog: -- 1 -- 2 -- 3 -- 4 -- 5 -- 6 -- 7 -- 8 -- 9 -- 10 -- 11 -- 12 -- 13 -- 14
		//       init prefix            ^                         ^                      ^
		//                              |                         |                      |
		//                     first ID-bearing          wlogIDBearingOffset    actual log end offset
		//
		expectedWLogTail := testOffsetWLogTail
		nextRecordID := istructs.FirstUserRecordID
		collectionOffset := testOffsetWLogIDBearing
		expectedIDBearingOffset := testOffsetWLogIDBearing
		require.Equal(testOffsetInitialWLogTail, getLastWLogOffset(t, vit, currentWS))
		if kind == checkpointNoUserIDs {
			expectedWLogTail = testOffsetInitialWLogTail
			collectionOffset = testOffsetInitialWLogTail
			expectedIDBearingOffset = istructs.NullOffset
			events := 0
			require.NoError(appStructs.Events().ReadWLog(t.Context(), currentWS.WSID, istructs.FirstOffset, istructs.ReadToTheEnd,
				func(offset istructs.Offset, event istructs.IWLogEvent) error {
					defer event.Release()
					events++
					require.Equal(istructs.Offset(events), offset)
					require.False(checkpoints.IsIDBearingEvent(event))
					return nil
				}))
			require.Equal(3, events)
		} else {
			fillWLogWithNonIDBearingEvents(t, vit, currentWS, testOffsetFirstIDBearing-1)
			firstIDBearing := vit.PostWS(currentWS, "c.sys.CUD", cudBody1Doc)
			require.Equal(testOffsetFirstIDBearing, firstIDBearing.CurrentWLogOffset)
			require.Len(firstIDBearing.NewIDs, 1)
			require.Equal(istructs.RecordID(200001), firstIDBearing.NewIDs["1"])

			fillWLogWithNonIDBearingEvents(t, vit, currentWS, testOffsetWLogIDBearing-1)
			lastRecordIDBearing := vit.PostWS(currentWS, "c.sys.CUD", fourAppWorkspaceDocs)
			require.Equal(testOffsetWLogIDBearing, lastRecordIDBearing.CurrentWLogOffset)
			require.Len(lastRecordIDBearing.NewIDs, 4)
			for index := range 4 {
				require.Equal(istructs.RecordID(200002+index), lastRecordIDBearing.NewIDs[strconv.Itoa(index+1)])
			}
			nextRecordID = lastRecordIDBearing.NewIDs["4"] + 1

			if offset := checkpointTestCaseToOffset(kind); offset != istructs.NullOffset || kind == checkpointMixedTail {
				fillWLogWithNonIDBearingEvents(t, vit, currentWS, testOffsetCUDFree)
				singleton := vit.PostWS(currentWS, "c.sys.CUD", singletonCUD)
				require.Equal(testOffsetSingleton, singleton.CurrentWLogOffset)
				require.Empty(singleton.NewIDs)
				updated := vit.PostWS(currentWS, "c.sys.CUD", fmt.Sprintf(
					`{"cuds":[{"sys.ID":%d,"fields":{}}]}`, lastRecordIDBearing.NewIDs["1"]))
				require.Equal(testOffsetWLogTail, updated.CurrentWLogOffset)
				require.Empty(updated.NewIDs)
			} else {
				// checkpoint test case is not about offsets
				switch kind {
				case checkpointAtWLogTail:
					expectedWLogTail = testOffsetWLogIDBearing
				default:
					fillWLogWithNonIDBearingEvents(t, vit, currentWS, testOffsetWLogTail)
				}
			}
		}
		require.Equal(expectedWLogTail, getLastWLogOffset(t, vit, currentWS))
		waitForActualizerFlush(t, appStructs, partitionID)
		fillPLogWithNonIDBearingEvents(t, vit, appQName, partitionID, currentWS.Owner, testOffsetPLogTail)
		require.Equal(testOffsetPLogTail, getLastPLogOffset(t, vit, appQName, partitionID))
		vit.WaitForOffset(plogOffsets, testOffsetPLogTail)
		vit.WaitForOffset(collectionOffsets, collectionOffset)

		expected = expectedWorkspaceNumbers{
			wsid:                currentWS.WSID,
			wlogIDBearingOffset: expectedIDBearingOffset,
			nextWLogOffset:      expectedWLogTail + 1,
			nextRecordID:        nextRecordID,
		}
	}
	return &recoveryCheckpointFixture{
		t: t, vit: vit, workspace: currentWS, appStructs: appStructs,
		partitionID: partitionID, expected: expected, logs: logCap, scope: scope, kind: kind,
	}
}

func (f *recoveryCheckpointFixture) restart() {
	f.t.Helper()
	require := require.New(f.t)
	appQName := f.workspace.AppQName()
	require.NoError(f.vit.RestartVVM(it.WithAfterStop(func(storage it.RestartStorage) {
		moveActualizerToPLogEnd(f.t, f.appStructs, f.partitionID)
		checkpointAppStorage := storage.AppStorage(f.t, istructs.AppQName_sys_vvm)
		checkpointStorage := vvmstorage.NewRecoveryCheckpointStorage(checkpointAppStorage)
		setCheckpoint(f.t, checkpointStorage, checkpointAppStorage,
			storage.AppStorage(f.t, appQName), istructs.ClusterApps[appQName], f.partitionID,
			f.expected.wsid, f.scope, f.kind)
		f.logs.Reset()
	})))
	f.workspace = f.vit.WS(appQName, recoveryTestWorkspaceName)
	require.Equal(f.expected.wsid, f.workspace.WSID)
}

func (f *recoveryCheckpointFixture) recoverPartition() {
	f.t.Helper()
	require := require.New(f.t)
	response := f.vit.PostWS(f.workspace, "c.sys.CUD", singleDoc,
		httpu.WithNoRetryPolicy(), httpu.Expect503())
	require.Equal(http.StatusServiceUnavailable, response.HTTPResp.StatusCode)
	f.logs.EventuallyHasLine("stage=cp.partition_recovery.complete", fmt.Sprintf("partid=%d", f.partitionID))
}

func (f *recoveryCheckpointFixture) requirePLogTail(wlogOffset istructs.Offset) {
	f.t.Helper()
	checkLastPLogEvent(f.t, f.vit, f.workspace.AppQName(), f.partitionID,
		f.workspace.WSID, testOffsetPLogTail+1, wlogOffset)
}

func (f *recoveryCheckpointFixture) requireCheckpointLogs() {
	f.t.Helper()
	checkRecoveryCheckpointLogs(f.t, f.logs, f.scope, f.kind)
}

// case: ws-creation projectors are done but the actualizer offset is not flushed yet
// on restart these projectors could be fired again
func waitForActualizerFlush(t *testing.T, appStructs istructs.IAppStructs, partitionID istructs.PartitionID) {
	t.Helper()
	projectors := []struct {
		name            appdef.QName
		recordQName     appdef.QName
		lastEventOffset istructs.Offset
	}{
		{name: appdef.NewQName(appdef.SysPackage, "InvokeCreateWorkspace"), recordQName: workspace.QNameCDocWorkspaceID},
		{name: appdef.NewQName(appdef.SysPackage, "InitializeWorkspace"), recordQName: appdef.QNameCDocWorkspaceDescriptor},
	}
	err := appStructs.Events().ReadPLog(t.Context(), partitionID, istructs.FirstOffset, istructs.ReadToTheEnd,
		func(offset istructs.Offset, event istructs.IPLogEvent) error {
			defer event.Release()
			for cud := range event.CUDs {
				if !cud.IsNew() {
					continue
				}
				for index := range projectors {
					if cud.QName() == projectors[index].recordQName {
						projectors[index].lastEventOffset = offset
					}
				}
			}
			return nil
		})
	require.NoError(t, err)
	// Both creation and initialization must have durable progress before restart.
	// Replaying creation can consume argument IDs; replaying initialization can
	// append two descriptor updates and advance the workspace's WLog offset.
	for _, projector := range projectors {
		if projector.lastEventOffset == istructs.NullOffset {
			continue
		}
		require.Eventually(t, func() bool {
			offset, err := actualizers.ActualizerOffset(appStructs, partitionID, projector.name)
			return err == nil && offset >= projector.lastEventOffset
		}, 10*time.Second, 10*time.Millisecond,
			"projector %s in partition %d must persist offset %d", projector.name, partitionID, projector.lastEventOffset)
	}
}

func moveActualizerToPLogEnd(t *testing.T, appStructs istructs.IAppStructs, partitionID istructs.PartitionID) {
	t.Helper()
	view := appdefsys.ProjectionOffsetsView
	key := appStructs.ViewRecords().KeyBuilder(view.Name)
	key.PutInt32(view.Fields.Partition, int32(partitionID))
	key.PutQName(view.Fields.Projector, checkpoints.QNameApplyRecoveryCheckpoint)
	value := appStructs.ViewRecords().NewValueBuilder(view.Name)
	value.PutInt64(view.Fields.Offset, int64(testOffsetPLogTail))
	require.NoError(t, appStructs.ViewRecords().Put(istructs.NullWSID, key, value))
}

func fillWLogWithNonIDBearingEvents(t *testing.T, vit *it.VIT, workspace *it.AppWorkspace, tail istructs.Offset) {
	t.Helper()
	current := getLastWLogOffset(t, vit, workspace)
	require.LessOrEqual(t, current, tail)
	for current < tail {
		response := vit.PostWS(workspace, nonIDBearingCommand, "{}")
		current++
		require.Equal(t, current, response.CurrentWLogOffset)
		require.Empty(t, response.NewIDs)
	}
}

func fillPLogWithNonIDBearingEvents(t *testing.T, vit *it.VIT, appQName appdef.AppQName,
	partitionID istructs.PartitionID, principal *it.Principal, tail istructs.Offset) {
	t.Helper()
	current := getLastPLogOffset(t, vit, appQName, partitionID)
	require.LessOrEqual(t, current, tail)
	for current < tail {
		response := vit.PostProfile(principal, nonIDBearingCommand, "{}")
		require.Empty(t, response.NewIDs)
		current++
	}
}

func recoveryVITConfig() it.VITConfig {
	appQName := istructs.AppQName_test1_app1
	return it.NewOwnVITConfig(it.WithApp(appQName, provideTestAppWithPartitions(3),
		it.WithUserLogin(recoveryTestLoginName, recoveryTestLoginPassword),
	))
}

type recoveryStorageObservers struct {
	checkpointWrites *sysVVMWriteRecorder
	appLogWrites     *logWriteRecorder
}

func recoveryVITConfigWithStorageHooks(errInjector *pLogReadErrorInjector, withUser bool, observers ...recoveryStorageObservers) it.VITConfig {
	appQName := istructs.AppQName_test1_app1
	var observation recoveryStorageObservers
	if len(observers) > 0 {
		observation = observers[0]
	}
	var options []it.AppOptFunc
	if withUser {
		options = append(options, it.WithUserLogin(recoveryTestLoginName, recoveryTestLoginPassword))
	}
	return it.NewOwnVITConfig(
		it.WithApp(appQName, provideRecoveryTestAppWithJobs, options...),
		it.WithVVMConfig(func(cfg *vvm.VVMConfig) {
			underlyingFactory := cfg.StorageFactory
			cfg.StorageFactory = func(currentTime timeu.ITime) (istorage.IAppStorageFactory, error) {
				underlying, err := underlyingFactory(currentTime)
				if err != nil {
					return nil, err
				}
				return &readErrorInjectingAndWriteRecordingStorageFactory{
					IAppStorageFactory: underlying,
					errInjector:        errInjector,
					sysVVMRecorder:     observation.checkpointWrites,
					appLogRecorder:     observation.appLogWrites,
				}, nil
			}
		}),
	)
}

func requireChangedPartitionCountRestartRejected(t *testing.T, vit *it.VIT, writes *logWriteRecorder) {
	t.Helper()
	appQName := istructs.AppQName_test1_app1
	storedDescriptor := deploymentDescriptorFields(t, vit.IAppStructsProvider, appQName)
	replacement := it.NewOwnVITConfig(it.WithApp(appQName,
		func(apis builtinapps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) builtinapps.Def {
			def := provideRecoveryTestAppWithJobs(apis, cfg, ep)
			def.NumParts = 2
			return def
		}))
	// When VAOperator restarts the VVM with a configured descriptor declaring NumPartitions 2
	// Then the VVM launch panics because the descriptor partition count has changed
	require.PanicsWithValue(t, "failed to deploy app test1/app1: status 409, expected [200 201]: num partitions changed: app test1/app1 declaring NumPartitions=2 but was previously deployed with NumPartitions=3", func() {
		require.NoError(t, vit.RestartVVM(it.WithNewConfig(&replacement), it.WithAfterStop(func(_ it.RestartStorage) {
			writes.reset()
		})))
	})
	// And the persisted deployment descriptor still has NumPartitions 3
	requireDeploymentDescriptorCount(t, vit.IAppStructsProvider, 3)
	require.Equal(t, storedDescriptor, deploymentDescriptorFields(t, vit.IAppStructsProvider, appQName))
	// And the application is not started with a different workspace placement
	_, err := vit.IAppPartitions.AppPartsCount(appQName)
	require.ErrorIs(t, err, appparts.ErrNotFound)
	writes.requireNoLogWrites(t)
}

func requireDeploymentDescriptorCount(t *testing.T, asp istructs.IAppStructsProvider, count istructs.NumAppPartitions) {
	t.Helper()
	descriptor, exists := findAppDeploymentDescriptor(t, asp, istructs.AppQName_test1_app1)
	require.True(t, exists)
	require.EqualValues(t, count, descriptor.AsInt32(cluster.Field_NumPartitions))
}

type recordIDBeforeRestart struct {
	wsid     istructs.WSID
	recordID istructs.RecordID
}

func createRecord(t *testing.T, vit *it.VIT) recordIDBeforeRestart {
	t.Helper()
	appQName := istructs.AppQName_test1_app1
	app, err := vit.IAppStructsProvider.BuiltIn(appQName)
	require.NoError(t, err)
	// Workspace 10 is the scenario's logical example. API v1 translates literal
	// WSID 10 as a pseudo-workspace, so use a real WSID with the same routing
	// boundary: partition 1 with count 3, but partition 0 with count 2.
	wsid := istructs.NewWSID(istructs.CurrentClusterID(), istructs.FirstBaseAppWSID+4)
	require.Equal(t, istructs.PartitionID(1), coreutils.AppPartitionID(wsid, 3))
	require.Equal(t, istructs.PartitionID(0), coreutils.AppPartitionID(wsid, 2), "reduced count would change routing")
	inited, err := cluster.InitAppWS(app, 1, wsid, getLastPLogOffset(t, vit, appQName, 1)+1,
		istructs.FirstOffset, istructs.UnixMilli(vit.Time.Now().UnixMilli()))
	require.NoError(t, err)
	require.True(t, inited)
	response := vit.PostApp(appQName, wsid, "c.sys.CUD", `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.DocBLOB"}}]}`,
		httpu.WithAuthorizeBy(vit.GetSystemPrincipal(appQName).Token))
	return recordIDBeforeRestart{wsid: wsid, recordID: response.NewID()}
}

func checkCorrectWSAfterCheckpointFailure(t *testing.T, vit *it.VIT, history recordIDBeforeRestart,
	writes *logWriteRecorder, logs logger.ILogCaptor) {
	t.Helper()
	appQName := istructs.AppQName_test1_app1
	wsid := history.wsid
	// Verify the descriptor count and workspace mapping as supporting routing checks.
	requireDeploymentDescriptorCount(t, vit.IAppStructsProvider, 3)
	count, err := vit.IAppPartitions.AppPartsCount(appQName)
	require.NoError(t, err)
	require.Equal(t, istructs.NumAppPartitions(3), count)
	partitionID, err := vit.IAppPartitions.AppWorkspacePartitionID(appQName, wsid)
	require.NoError(t, err)
	require.Equal(t, istructs.PartitionID(1), partitionID)
	writes.reset()
	logs.Reset()
	// When a client submits a command to workspace 10 through the router
	response := vit.PostApp(appQName, wsid, "c.sys.CUD",
		fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":%d,"sys.IsActive":false}}]}`, history.recordID),
		httpu.WithAuthorizeBy(vit.GetSystemPrincipal(appQName).Token), httpu.WithNoRetryPolicy(), httpu.Expect500())
	// Then the client receives "500 Internal Server Error"
	require.Equal(t, http.StatusInternalServerError, response.HTTPResp.StatusCode)
	// And response error is "partition not found"
	requirePartitionNotFound(t, response.SysError)
	// Verify that the failed request is not rerouted and performs no recovery or writes.
	logs.HasLine("stage=cp.error", "vapp=test1/app1", "partid=1", fmt.Sprintf("wsid=%d", wsid), "partition not found")
	logs.NotContains("stage=cp.partition_recovery.start", "stage=cp.workspace_recovery.start")
	writes.requireNoLogWrites(t)
	// Verify that the failed request leaves existing workspace data unchanged.
	app, err := vit.IAppStructsProvider.BuiltIn(appQName)
	require.NoError(t, err)
	record, err := app.Records().Get(wsid, true, history.recordID)
	require.NoError(t, err)
	require.Equal(t, appdef.NewQName("app1pkg", "DocBLOB"), record.QName())
	require.True(t, record.AsBool(appdef.SystemField_IsActive))
}

func provideRecoveryTestAppWithJobs(apis builtinapps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) builtinapps.Def {
	def := provideTestAppWithPartitions(3)(apis, cfg, ep)
	def.NumAppWorkspaces = 3 // Give every partition an app workspace and scheduled job.
	schema, err := it.SchemaTestApp1FS.ReadFile("schemaTestApp1.vsql")
	if err != nil {
		panic(err)
	}
	for index := range def.Packages {
		if def.Packages[index].Path == it.App1PkgPath {
			def.Packages[index].FS = fstest.MapFS{
				"schemaTestApp1.vsql": &fstest.MapFile{Data: schema},
				"initial_recovery_job.vsql": &fstest.MapFile{Data: []byte(`ALTER WORKSPACE sys.AppWorkspaceWS (
					EXTENSION ENGINE BUILTIN (JOB InitialRecoveryJob '@every 1h');
				);`)},
			}
		}
	}
	cfg.AddJobs(istructsmem.BuiltinJob{Name: appdef.NewQName("app1pkg", "InitialRecoveryJob"),
		Func: func(istructs.IState, istructs.IIntents) error { return nil }})
	return def
}

func requirePartitionCheckpointPresence(t *testing.T, vit *it.VIT, failed []istructs.PartitionID) {
	t.Helper()
	store, err := vit.IAppStorageProvider.AppStorage(istructs.AppQName_sys_vvm)
	require.NoError(t, err)
	checkpointStorage := vvmstorage.NewRecoveryCheckpointStorage(store)
	for id := range istructs.PartitionID(3) {
		_, present, err := checkpointStorage.GetPartitionCheckpoint(istructs.ClusterApps[istructs.AppQName_test1_app1], id)
		require.NoError(t, err)
		require.Equal(t, !containsPartition(failed, id), present)
	}
}

func requireInitialRecoveryFailureLogs(t *testing.T, logs logger.ILogCaptor, failed []istructs.PartitionID) {
	t.Helper()
	for _, id := range failed {
		logs.HasLine("stage=bootstrap.initial_recovery.error", "app=test1/app1",
			fmt.Sprintf("partid=%d", id), "read PLog", "PLog read failed")
	}
}

func removeAppPartitionCheckpoints(t *testing.T, storage it.RestartStorage, numPartitions istructs.NumAppPartitions) {
	t.Helper()
	store := storage.AppStorage(t, istructs.AppQName_sys_vvm)
	checkpointsStorage := vvmstorage.NewRecoveryCheckpointStorage(store)
	for id := istructs.PartitionID(0); id < istructs.PartitionID(numPartitions); id++ {
		deletePartitionCheckpoint(t, checkpointsStorage, store, istructs.ClusterApps[istructs.AppQName_test1_app1], id)
	}
}

func deployedPartitionIDs(t *testing.T, vit *it.VIT) []istructs.PartitionID {
	t.Helper()
	var deployed []istructs.PartitionID
	for id := range istructs.PartitionID(3) {
		partition, err := vit.IAppPartitions.Borrow(istructs.AppQName_test1_app1, id, appparts.ProcessorKind_Command)
		if errors.Is(err, appparts.ErrNotFound) {
			continue
		}
		require.NoError(t, err)
		partition.Release()
		deployed = append(deployed, id)
	}
	return deployed
}

func requirePartitionNotFound(t *testing.T, err error) {
	t.Helper()
	var responseError coreutils.SysError
	require.ErrorAs(t, err, &responseError)
	require.Equal(t, http.StatusInternalServerError, responseError.HTTPStatus)
	require.Equal(t, "partition not found", responseError.Message)
}

func containsPartition(partitions []istructs.PartitionID, target istructs.PartitionID) bool {
	return slices.Contains(partitions, target)
}

type readErrorInjectingAndWriteRecordingStorageFactory struct {
	istorage.IAppStorageFactory
	errInjector      *pLogReadErrorInjector
	appStoragePrefix string // Defaults to test1app1; selects the application whose PLog reads can fail.
	sysVVMRecorder   *sysVVMWriteRecorder
	appLogRecorder   *logWriteRecorder
}

func (f *readErrorInjectingAndWriteRecordingStorageFactory) AppStorage(appName istorage.SafeAppName) (istorage.IAppStorage, error) {
	storage, err := f.IAppStorageFactory.AppStorage(appName)
	if err == nil && f.sysVVMRecorder != nil && strings.HasPrefix(appName.String(), "sysvvm") {
		return &sysVVMWriteRecorderStorage{IAppStorage: storage, writes: f.sysVVMRecorder}, nil
	}
	prefix := f.appStoragePrefix
	if prefix == "" {
		prefix = "test1app1"
	}
	if err != nil || !strings.HasPrefix(appName.String(), prefix) {
		return storage, err
	}
	if f.appLogRecorder != nil {
		storage = &appLogWriteRecorderStorage{IAppStorage: storage, logRecorder: f.appLogRecorder}
	}
	return &pLogReadFailureStorage{IAppStorage: storage, errInjector: f.errInjector}, nil
}

type logWriteKind string

const (
	pLogWrite logWriteKind = "PLog"
	wLogWrite logWriteKind = "WLog"
)

type logWriteRecorder struct {
	mu         sync.Mutex
	pLogPrefix []byte
	wLogPrefix []byte
	writes     []logWriteKind
}

func newAppLogWriteRecorder() *logWriteRecorder {
	pLogKey, _ := istructsmem.PLogEventKey(0, istructs.FirstOffset)
	wLogKey, _ := istructsmem.WLogEventKey(0, istructs.FirstOffset)
	// Both key families begin with the two-byte system view identifier. Match
	// that prefix to observe writes for every partition, workspace, and offset.
	return &logWriteRecorder{pLogPrefix: bytes.Clone(pLogKey[:2]), wLogPrefix: bytes.Clone(wLogKey[:2])}
}

func (r *logWriteRecorder) record(key []byte) {
	var kind logWriteKind
	switch {
	case bytes.HasPrefix(key, r.pLogPrefix):
		kind = pLogWrite
	case bytes.HasPrefix(key, r.wLogPrefix):
		kind = wLogWrite
	default:
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writes = append(r.writes, kind)
}

func (r *logWriteRecorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writes = nil
}

func (r *logWriteRecorder) requireLogWrites(t *testing.T) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	require.Contains(t, r.writes, pLogWrite)
	require.Contains(t, r.writes, wLogWrite)
}

func (r *logWriteRecorder) requireNoLogWrites(t *testing.T) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	require.NotContains(t, r.writes, pLogWrite, "no PLog write may be attempted in any partition")
	require.NotContains(t, r.writes, wLogWrite, "no WLog write may be attempted in any workspace")
}

type appLogWriteRecorderStorage struct {
	istorage.IAppStorage
	logRecorder *logWriteRecorder
}

func (s *appLogWriteRecorderStorage) Put(key, columns, value []byte) error {
	s.logRecorder.record(key)
	return s.IAppStorage.Put(key, columns, value)
}

func (s *appLogWriteRecorderStorage) InsertIfNotExists(key, columns, value []byte, ttlSeconds int) (bool, error) {
	s.logRecorder.record(key)
	return s.IAppStorage.InsertIfNotExists(key, columns, value, ttlSeconds)
}

func (s *appLogWriteRecorderStorage) PutBatch(items []istorage.BatchItem) error {
	for _, item := range items {
		s.logRecorder.record(item.PKey)
	}
	return s.IAppStorage.PutBatch(items)
}

type pLogReadFailureStorage struct {
	istorage.IAppStorage
	errInjector *pLogReadErrorInjector
}

func (s *pLogReadFailureStorage) Read(ctx context.Context, pKey, startCCols, finishCCols []byte,
	callback istorage.ReadCallback) error {
	if err := s.errInjector.shouldFailRead(pKey); err != nil {
		return err
	}
	return s.IAppStorage.Read(ctx, pKey, startCCols, finishCCols, callback)
}

// pLogReadErrorInjector controls only the storage dependency:
//
//	failPLogReads(ids) -> selected PLog keys return an error
//	                  -> unrelated reads use real storage
//	resolve()         -> all reads use real storage
//
// Resolving storage does not deploy excluded partitions; only a later bootstrap can.
type pLogReadErrorInjector struct {
	mu                  sync.Mutex
	plogEventKeysToFail map[string]bool // reads by these keys will be failed
}

func newPLogReadFailure() *pLogReadErrorInjector {
	return &pLogReadErrorInjector{plogEventKeysToFail: map[string]bool{}}
}

func (c *pLogReadErrorInjector) failPLogReads(ids []istructs.PartitionID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range ids {
		key, _ := istructsmem.PLogEventKey(id, istructs.FirstOffset)
		c.plogEventKeysToFail[string(key)] = true
	}
}

func (c *pLogReadErrorInjector) shouldFailRead(key []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.plogEventKeysToFail[string(key)] {
		return errors.New("PLog read failed")
	}
	return nil
}

// should not be named e.g. reset() since `resolve` term is in Gherkin scenarios
func (c *pLogReadErrorInjector) resolve() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.plogEventKeysToFail)
}

// Record successful sys/vvm writes so tests can check checkpoint persistence order.
type sysVVMWriteRecorderStorage struct {
	istorage.IAppStorage
	writes *sysVVMWriteRecorder
}

func (s *sysVVMWriteRecorderStorage) Put(key, columns, value []byte) error {
	if err := s.IAppStorage.Put(key, columns, value); err != nil {
		return err
	}
	s.writes.mu.Lock()
	defer s.writes.mu.Unlock()
	s.writes.keys = append(s.writes.keys, bytes.Clone(key))
	return nil
}

type sysVVMWriteRecorder struct {
	mu   sync.Mutex
	keys [][]byte
}

func createWorkspacesInPartition(t *testing.T, vit *it.VIT, owner *it.Principal,
	partitionID istructs.PartitionID, count int) []*it.AppWorkspace {
	t.Helper()
	result := make([]*it.AppWorkspace, 0, count)
	for candidate := 0; len(result) < count; candidate++ {
		workspace := vit.CreateWorkspace(it.SimpleWSParams(fmt.Sprintf("initial-recovery-%d-%d", partitionID, candidate)), owner)
		actualPartitionID, err := vit.IAppPartitions.AppWorkspacePartitionID(workspace.AppQName(), workspace.WSID)
		require.NoError(t, err)
		if actualPartitionID == partitionID {
			result = append(result, workspace)
		}
	}
	return result
}

func deletePartitionCheckpoint(t *testing.T, checkpointStorage checkpoints.IRecoveryCheckpointStorage,
	storage istorage.IAppStorage, appID istructs.ClusterAppID, partitionID istructs.PartitionID) {
	t.Helper()
	_, ok, err := checkpointStorage.GetPartitionCheckpoint(appID, partitionID)
	require.NoError(t, err)
	if !ok {
		return
	}
	pKey, cCols := vvmstorage.PartitionCheckpointKey(appID, partitionID)
	deleteFromStorage(t, storage, pKey, cCols)
}

func deleteWorkspaceCheckpoint(t *testing.T, checkpointStorage checkpoints.IRecoveryCheckpointStorage,
	storage istorage.IAppStorage, appID istructs.ClusterAppID, wsid istructs.WSID) {
	t.Helper()
	_, ok, err := checkpointStorage.GetWorkspaceCheckpoint(appID, wsid)
	require.NoError(t, err)
	if !ok {
		return
	}
	pKey, cCols := vvmstorage.WorkspaceCheckpointKey(appID, wsid)
	deleteFromStorage(t, storage, pKey, cCols)
}

func getLastPLogOffset(t *testing.T, vit *it.VIT, appQName appdef.AppQName, partitionID istructs.PartitionID) (lastPLogOffset istructs.Offset) {
	t.Helper()
	appStructs, err := vit.IAppStructsProvider.BuiltIn(appQName)
	require.NoError(t, err)
	err = appStructs.Events().ReadPLog(t.Context(), partitionID, istructs.FirstOffset, istructs.ReadToTheEnd,
		func(offset istructs.Offset, event istructs.IPLogEvent) error {
			defer event.Release()
			lastPLogOffset = offset
			return nil
		})
	require.NoError(t, err)
	return lastPLogOffset
}

func getLastWLogOffset(t *testing.T, vit *it.VIT, workspace *it.AppWorkspace) istructs.Offset {
	t.Helper()
	appStructs, err := vit.IAppStructsProvider.BuiltIn(workspace.AppQName())
	require.NoError(t, err)
	tail := istructs.NullOffset
	err = appStructs.Events().ReadWLog(t.Context(), workspace.WSID, istructs.FirstOffset, istructs.ReadToTheEnd,
		func(offset istructs.Offset, event istructs.IWLogEvent) error {
			defer event.Release()
			tail = offset
			return nil
		})
	require.NoError(t, err)
	return tail
}

func setCheckpoint(t *testing.T, checkpointStorage checkpoints.IRecoveryCheckpointStorage,
	checkpointAppStorage, appStorage istorage.IAppStorage, appID istructs.ClusterAppID,
	partitionID istructs.PartitionID, wsid istructs.WSID, chType checkpointType, kind checkpointTestCase) {
	t.Helper()

	if chType == checkpointTypePartition {
		setPartitionCheckpoint(t, checkpointStorage, checkpointAppStorage, appStorage, appID, partitionID, kind)
		return
	}

	require.NoError(t, checkpointStorage.PutPartitionCheckpoint(appID, partitionID, checkpoints.PartitionCheckpoint{LastPLogOffset: testOffsetPLogTail}))
	setWorkspaceCheckpoint(t, checkpointStorage, checkpointAppStorage, appStorage, appID, wsid, kind)
}

func setPartitionCheckpoint(t *testing.T, checkpointStorage checkpoints.IRecoveryCheckpointStorage,
	checkpointAppStorage, appStorage istorage.IAppStorage, appID istructs.ClusterAppID,
	partitionID istructs.PartitionID, kind checkpointTestCase) {
	t.Helper()
	require := require.New(t)
	var offset istructs.Offset
	switch kind {
	case checkpointAbsent:
		deletePartitionCheckpoint(t, checkpointStorage, checkpointAppStorage, appID, partitionID)
		_, ok, err := checkpointStorage.GetPartitionCheckpoint(appID, partitionID)
		require.NoError(err)
		require.False(ok)
		return
	case checkpointZero:
		offset = istructs.NullOffset
	case checkpointValidButOutdated:
		offset = 10
	case checkpointValid:
		offset = testOffsetPLogTail
	case checkpointBeyondTail:
		offset = testOffsetBeyondPLogTail
	case checkpointMissingEvent, checkpointBeforeBeginning:
		offset = 10
		if kind == checkpointBeforeBeginning {
			offset = istructs.FirstOffset
		}
		pKey, cCols := istructsmem.PLogEventKey(partitionID, offset)
		deleteFromStorage(t, appStorage, pKey, cCols)
	default:
		require.FailNow("unsupported partition checkpoint kind", "%d", kind)
	}
	require.NoError(checkpointStorage.PutPartitionCheckpoint(appID, partitionID,
		checkpoints.PartitionCheckpoint{LastPLogOffset: offset}))
}

func setWorkspaceCheckpoint(t *testing.T, checkpointStorage checkpoints.IRecoveryCheckpointStorage,
	checkpointAppStorage, appStorage istorage.IAppStorage, appID istructs.ClusterAppID,
	wsid istructs.WSID, kind checkpointTestCase) {
	t.Helper()
	var offset istructs.Offset
	switch kind {
	case checkpointAbsent:
		pKey, cCols := vvmstorage.WorkspaceCheckpointKey(appID, wsid)
		_, exists, err := checkpointStorage.GetWorkspaceCheckpoint(appID, wsid)
		require.NoError(t, err)
		if exists {
			deleteFromStorage(t, checkpointAppStorage, pKey, cCols)
		}
		_, ok, err := checkpointStorage.GetWorkspaceCheckpoint(appID, wsid)
		require.NoError(t, err)
		require.False(t, ok)
		return
	case checkpointZero:
		offset = istructs.NullOffset
	case checkpointValidButOutdated:
		offset = testOffsetFirstIDBearing
	case checkpointValid, checkpointMixedTail:
		offset = testOffsetWLogIDBearing
	case checkpointAtWLogTail:
		offset = testOffsetWLogIDBearing
	default:
		offset = checkpointTestCaseToOffset(kind)
		if kind == checkpointMissingEvent {
			pKey, cCols := istructsmem.WLogEventKey(wsid, offset)
			deleteFromStorage(t, appStorage, pKey, cCols)
		}
	}
	require.NoError(t, checkpointStorage.PutWorkspaceCheckpoint(appID, wsid,
		checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: offset}))
}

func checkRecoveryCheckpointLogs(t *testing.T, logCap logger.ILogCaptor, chType checkpointType,
	testCase checkpointTestCase) {
	t.Helper()
	partitionCheckpointInvalid := testCase == checkpointBeyondTail || testCase == checkpointMissingEvent || testCase == checkpointBeforeBeginning
	if chType == checkpointTypePartition && partitionCheckpointInvalid {
		offset := testOffsetBeyondPLogTail
		switch testCase {
		case checkpointMissingEvent:
			offset = 10
		case checkpointBeforeBeginning:
			offset = istructs.FirstOffset
		}
		message := fmt.Sprintf(
			"stored PLog offset %d does not identify an existing event; retrying from FirstOffset", offset)
		logCap.HasLine("level=ERROR", "stage=cp.partition_recovery.checkpoint.invalid",
			fmt.Sprintf("msg=%q", message))
	} else {
		logCap.NotContains("cp.partition_recovery.checkpoint.invalid")
	}

	offset := checkpointTestCaseToOffset(testCase)
	if chType == checkpointTypeWorkspace && offset > istructs.NullOffset {
		message := fmt.Sprintf(
			"stored WLog offset %d does not identify an ID-bearing event; retrying from FirstOffset", offset)
		logCap.HasLine("level=ERROR", "stage=cp.workspace_recovery.checkpoint.invalid",
			fmt.Sprintf("msg=%q", message))
	} else {
		logCap.NotContains("cp.workspace_recovery.checkpoint.invalid")
	}
}

func checkpointTestCaseToOffset(testCase checkpointTestCase) (offset istructs.Offset) {
	switch testCase {
	case checkpointNoUserIDs:
		return istructs.FirstOffset
	case checkpointBeyondTail:
		return testOffsetWLogTail + 1
	case checkpointMissingEvent:
		return testOffsetMissingWLog
	case checkpointCUDFree:
		return testOffsetCUDFree
	case checkpointSingleton:
		return testOffsetSingleton
	case checkpointExistingRecordUpdate:
		return testOffsetWLogTail
	default:
		// cases when the checkpoint test case is not about an offset
		// are handled manually
		return istructs.NullOffset
	}
}

func deleteFromStorage(t *testing.T, storage istorage.IAppStorage, pKey, cCols []byte) {
	t.Helper()
	var value []byte
	ok, err := storage.Get(pKey, cCols, &value)
	require.NoError(t, err)
	require.True(t, ok, "storage cell to delete does not exist")
	ok, err = storage.CompareAndDelete(pKey, cCols, value)
	require.NoError(t, err)
	require.True(t, ok, "storage cell changed before deletion")
}

func provideTestApp(apis builtinapps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) builtinapps.Def {
	return provideTestAppWithPartitions(1)(apis, cfg, ep)
}

func provideTestAppWithPartitions(numPartitions istructs.NumAppPartitions) builtinapps.Builder {
	return func(apis builtinapps.APIs, cfg *istructsmem.AppConfigType, ep extensionpoints.IExtensionPoint) builtinapps.Def {
		def := it.ProvideApp1(apis, cfg, ep)
		def.NumParts = numPartitions
		def.NumAppWorkspaces = 1
		return def
	}
}

func checkNextWorkspaceSequence(vit *it.VIT, workspace *it.AppWorkspace,
	expected expectedWorkspaceNumbers) istructs.Offset {
	require := require.New(vit.T)

	// insert a test doc and check its counters
	resp := vit.PostWS(workspace, "c.sys.CUD", singleDoc)
	require.Len(resp.NewIDs, 1)
	require.Equal(expected.nextRecordID, resp.NewIDs["1"], "unexpected NewID in response at WLog offset %d", resp.CurrentWLogOffset)
	require.Equal(expected.nextWLogOffset, resp.CurrentWLogOffset, "unexpected WLog offset for NewID %d", resp.NewIDs["1"])

	// check the CurrentWLogOffset is the actual WLog end
	appStructs, err := vit.IAppStructsProvider.BuiltIn(workspace.AppQName())
	require.NoError(err)
	eventCount := 0
	err = appStructs.Events().ReadWLog(vit.T.Context(), workspace.WSID, resp.CurrentWLogOffset, istructs.ReadToTheEnd,
		func(wlogOffset istructs.Offset, event istructs.IWLogEvent) error {
			defer event.Release()
			eventCount++
			require.Equal(resp.CurrentWLogOffset, wlogOffset)
			cudCount := 0
			for cud := range event.CUDs {
				cudCount++
				require.Equal(resp.NewIDs["1"], cud.ID())
			}
			require.Equal(1, cudCount)
			return nil
		})
	require.NoError(err)
	require.Equal(1, eventCount, "unexpected events after WLog offset %d", resp.CurrentWLogOffset)
	return resp.CurrentWLogOffset
}

func checkLastPLogEvent(t *testing.T, vit *it.VIT, appQName appdef.AppQName,
	partitionID istructs.PartitionID, wsid istructs.WSID, expectedLastPLogOffset, expectedWLogOffset istructs.Offset) {
	t.Helper()
	appStructs, err := vit.IAppStructsProvider.BuiltIn(appQName)
	require.NoError(t, err)
	eventCount := 0
	err = appStructs.Events().ReadPLog(t.Context(), partitionID, expectedLastPLogOffset, istructs.ReadToTheEnd,
		func(offset istructs.Offset, event istructs.IPLogEvent) error {
			defer event.Release()
			eventCount++
			require.Equal(t, expectedLastPLogOffset, offset)
			require.Equal(t, wsid, event.Workspace())
			require.Equal(t, expectedWLogOffset, event.WLogOffset())
			return nil
		})
	require.NoError(t, err)
	require.Equal(t, 1, eventCount, "unexpected events after PLog offset %d", expectedLastPLogOffset)
}

func checkWorkspaceCheckpoint(t *testing.T, checkpointStorage checkpoints.IRecoveryCheckpointStorage,
	appID istructs.ClusterAppID, expected expectedWorkspaceNumbers) {
	t.Helper()
	checkpoint, ok, err := checkpointStorage.GetWorkspaceCheckpoint(appID, expected.wsid)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, expected.wlogIDBearingOffset, checkpoint.WLogIDBearingOffset)
}
