/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package commandprocessor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/constraints"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/checkpoints"
	"github.com/voedger/voedger/pkg/checkpoints/testutils"
	"github.com/voedger/voedger/pkg/coreutils"
	wsdescutil "github.com/voedger/voedger/pkg/coreutils/testwsdesc"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/iauthnzimpl"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/in10nmem"
	"github.com/voedger/voedger/pkg/iratesce"
	"github.com/voedger/voedger/pkg/isecretsimpl"
	"github.com/voedger/voedger/pkg/istorage/mem"
	istorageimpl "github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/processors"
	"github.com/voedger/voedger/pkg/processors/actualizers"
	"github.com/voedger/voedger/pkg/vvm/engines"
)

var (
	testCRecord      = appdef.NewQName("test", "TestCRecord")
	testCDoc         = appdef.NewQName("test", "TestCDoc")
	testWDoc         = appdef.NewQName("test", "TestWDoc")
	testSingleton    = appdef.NewQName("test", "TestSingleton")
	testNoCUDCommand = appdef.NewQName(appdef.SysPackage, "TestNoCUD")
	testNoCUDParams  = appdef.NewQName(appdef.SysPackage, "TestNoCUDParams")
)

func TestBasicUsage(t *testing.T) {
	require := require.New(t)
	check := make(chan interface{}, 1)

	testCmdQName := appdef.NewQName(appdef.SysPackage, "Test")
	// schema of parameters of the test command
	testCmdQNameParams := appdef.NewQName(appdef.SysPackage, "TestParams")
	// schema of unlogged parameters of the test command
	testCmdQNameParamsUnlogged := appdef.NewQName(appdef.SysPackage, "TestParamsUnlogged")
	prepareWS := func(wsb appdef.IWorkspaceBuilder, cfg *istructsmem.AppConfigType) {
		pars := wsb.AddObject(testCmdQNameParams)
		pars.AddField("Text", appdef.DataKind_string, true)

		unloggedPars := wsb.AddObject(testCmdQNameParamsUnlogged)
		unloggedPars.AddField("Password", appdef.DataKind_string, true)

		wsb.AddCDoc(testCDoc).AddContainer("TestCRecord", testCRecord, 0, 1)
		wsb.AddCRecord(testCRecord)

		wsb.AddCommand(testCmdQName).SetUnloggedParam(testCmdQNameParamsUnlogged).SetParam(testCmdQNameParams)

		addStandardTestRoles(wsb)

		// the test command itself
		testExec := func(args istructs.ExecCommandArgs) (err error) {
			require.Equal(istructs.WSID(1), args.PrepareArgs.WSID)
			require.NotNil(args.State)

			// check that we received exactly what client sent
			text := args.ArgumentObject.AsString("Text")
			if text == "fire error" {
				return errors.New(text)
			} else {
				require.Equal("hello", text)
			}
			require.Equal("pass", args.ArgumentUnloggedObject.AsString("Password"))

			check <- 1 // signal that the checking is done
			return
		}
		testCmd := istructsmem.NewCommandFunction(testCmdQName, testExec)
		cfg.Resources.Add(testCmd)
	}

	logCap := logger.StartCapture(t, logger.LogLevelVerbose)

	app := setUp(t, prepareWS)
	defer tearDown(app)

	channelID, channelCleanup, err := app.n10nBroker.NewChannel("test", 24*time.Hour)
	require.NoError(err)
	defer channelCleanup()
	projectionKey := in10n.ProjectionKey{
		App:        istructs.AppQName_untill_airs_bp,
		Projection: actualizers.PLogUpdatesQName,
		WS:         1,
	}
	go app.n10nBroker.WatchChannel(app.ctx, channelID, func(projection in10n.ProjectionKey, _ istructs.Offset) {
		require.Equal(projectionKey, projection)
		check <- 1
	})
	err = app.n10nBroker.Subscribe(channelID, projectionKey)
	require.NoError(err)
	defer func() { require.NoError(app.n10nBroker.Unsubscribe(channelID, projectionKey)) }()

	t.Run("basic usage", func(t *testing.T) {
		// command processor works through ibus.SendResponse -> we need a sender -> let's test using ibus.SendRequest2()
		request := bus.Request{
			Body:     []byte(`{"args":{"Text":"hello"},"unloggedArgs":{"Password":"pass"}}`),
			AppQName: istructs.AppQName_untill_airs_bp,
			WSID:     1,
			Resource: "c.sys.Test",
			// need to authorize, otherwise execute will be forbidden
			Header: app.sysAuthHeader,
		}

		respCh, respMeta, respErr, err := app.requestSender.SendRequest(app.ctx, request)
		require.NoError(err)
		respData := ""
		for elem := range respCh {
			require.Empty(respData)
			respDataBytes, err := json.Marshal(elem)
			require.NoError(err)
			respData = string(respDataBytes)
		}
		require.NoError(*respErr)
		log.Println(respData)
		require.Equal(http.StatusOK, respMeta.StatusCode)
		require.Equal(httpu.ContentType_ApplicationJSON, respMeta.ContentType)
		// check that command is handled and notifications were sent
		<-check
		<-check

		logCap.EventuallyHasLine("stage=cp.success")
	})

	t.Run("500 internal server error command exec error", func(t *testing.T) {
		logCap.Reset()
		request := bus.Request{
			Body:     []byte(`{"args":{"Text":"fire error"},"unloggedArgs":{"Password":"pass"}}`),
			AppQName: istructs.AppQName_untill_airs_bp,
			WSID:     1,
			Resource: "c.sys.Test",
			Header:   app.sysAuthHeader,
		}
		respCh, respMeta, respErr, err := app.requestSender.SendRequest(app.ctx, request)
		require.NoError(err)
		counter := 0
		for elem := range respCh {
			require.Zero(counter)
			require.Equal(http.StatusInternalServerError, respMeta.StatusCode)
			require.Equal(httpu.ContentType_ApplicationJSON, respMeta.ContentType)
			require.Equal(coreutils.SysError{HTTPStatus: http.StatusInternalServerError, Message: "fire error"}, elem) // nolint errorlint
			counter++
		}
		require.NoError(*respErr)

		logCap.HasLine("stage=cp.error", "fire error")
	})
}

func sendCUD_3Inserts(t *testing.T, wsid istructs.WSID, app testApp, expectedCode ...int) map[string]interface{} {
	return sendCUD_3Inserts_WithSender(t, wsid, app, app.requestSender, expectedCode...)
}

func sendCUD_3Inserts_WithSender(t *testing.T, wsid istructs.WSID, app testApp, sender bus.IRequestSender, expectedCode ...int) map[string]interface{} {
	return sendRequestWithSender(t, app, sender, newCUD_3Inserts_Request(wsid, app), expectedCode...)
}

func sendRequestWithSender(t *testing.T, app testApp, sender bus.IRequestSender, request bus.Request,
	expectedCode ...int) map[string]interface{} {
	require := require.New(t)
	respCh, respMeta, respErr, err := sender.SendRequest(app.ctx, request)
	require.NoError(err)
	respDataStr := ""
	for elem := range respCh {
		require.Empty(respDataStr)
		switch typed := elem.(type) {
		case string:
			respDataStr = typed
		case coreutils.SysError:
			respDataStr = typed.ToJSON_APIV1()
		}
	}
	if len(expectedCode) == 0 {
		require.Equal(http.StatusOK, respMeta.StatusCode, respDataStr)
	} else {
		require.Equal(expectedCode[0], respMeta.StatusCode, respDataStr)
	}

	respData := map[string]interface{}{}
	if len(respDataStr) > 0 {
		require.NoError(json.Unmarshal([]byte(respDataStr), &respData))
	}
	require.NoError(*respErr)
	return respData
}

func sendCUDWithBody(t *testing.T, wsid istructs.WSID, app testApp, body string) map[string]interface{} {
	request := newCUD_3Inserts_Request(wsid, app)
	request.Body = []byte(body)
	return sendRequestWithSender(t, app, app.requestSender, request)
}

func sendNoCUDCommand(t *testing.T, wsid istructs.WSID, app testApp) map[string]interface{} {
	return sendRequestWithSender(t, app, app.requestSender, bus.Request{
		WSID:     wsid,
		AppQName: istructs.AppQName_untill_airs_bp,
		Resource: "c.sys.TestNoCUD",
		Body:     []byte(`{"args":{"Marker":1}}`),
		Header:   app.sysAuthHeader,
	})
}

func newCUD_3Inserts_Request(wsid istructs.WSID, app testApp) bus.Request {
	return bus.Request{
		WSID:     wsid,
		AppQName: istructs.AppQName_untill_airs_bp,
		Resource: "c.sys.CUD",
		Body: []byte(`{"cuds":[
			{"fields":{"sys.ID":1,"sys.QName":"test.TestCDoc"}},
			{"fields":{"sys.ID":2,"sys.QName":"test.TestWDoc"}},
			{"fields":{"sys.ID":3,"sys.QName":"test.TestCRecord","sys.ParentID":1,"sys.Container":"TestCRecord"}}
		]}`),
		Header: app.sysAuthHeader,
	}
}

func sendRequestForStatus(ctx context.Context, sender bus.IRequestSender, request bus.Request) (int, error) {
	responseCh, responseMeta, responseErr, err := sender.SendRequest(ctx, request)
	if err != nil {
		return 0, err
	}
	for range responseCh {
	}
	return responseMeta.StatusCode, *responseErr
}

func partitionManagerKeyForWSID(wsid istructs.WSID) partitionKey {
	return partitionKey{
		appQName:    testAppName,
		partitionID: coreutils.AppPartitionID(wsid, testAppPartCount),
	}
}

func recoveryKeyForTestApp(app testApp, wsid istructs.WSID) partitionKey {
	return partitionKey{
		appQName:    testAppName,
		partitionID: coreutils.AppPartitionID(wsid, app.numAppPartitions),
	}
}

func recoveryWorkspaceKeyForTestApp(app testApp, wsid istructs.WSID) workspaceKey {
	return workspaceKey{partitionKey: recoveryKeyForTestApp(app, wsid), wsid: wsid}
}

func triggerAndWaitForRecovery(t *testing.T, app testApp, wsid istructs.WSID) error {
	t.Helper()
	sendCUD_3Inserts_WithSender(t, wsid, app, app.rawRequestSender, http.StatusServiceUnavailable)
	return app.recoveryTestControl.partitions.wait(app.ctx, partitionManagerKeyForWSID(wsid))
}

func triggerWorkspaceRecovery(t *testing.T, app testApp, wsid istructs.WSID) {
	t.Helper()
	sendCUD_3Inserts_WithSender(t, wsid, app, app.rawRequestSender, http.StatusServiceUnavailable)
}

func triggerAndWaitForWorkspaceRecovery(t *testing.T, app testApp, wsid istructs.WSID) error {
	t.Helper()
	if err := triggerAndWaitForRecovery(t, app, wsid); err != nil {
		return err
	}
	triggerWorkspaceRecovery(t, app, wsid)
	return app.recoveryTestControl.workspaces.wait(app.ctx, workspaceManagerKey(wsid))
}

func TestRecoveryOnSyncProjectorError(t *testing.T) {
	require := require.New(t)

	logCap := logger.StartCapture(t, logger.LogLevelVerbose)

	cudQName := appdef.NewQName(appdef.SysPackage, "CUD")
	testErr := errors.New("test error")
	failingProjQName := appdef.NewQName(appdef.SysPackage, "Failer")
	projExtension := "p." + failingProjQName.String()
	counter := 0
	app := setUp(t, func(wsb appdef.IWorkspaceBuilder, cfg *istructsmem.AppConfigType) {
		addTestCUDRecords(wsb)
		addTestCUDCommand(wsb, cfg)

		addStandardTestRoles(wsb)

		cfg.AddSyncProjectors(
			istructs.Projector{
				Name: failingProjQName,
				Func: func(istructs.IPLogEvent, istructs.IState, istructs.IIntents) error {
					counter++
					if counter == 3 { // 1st event is insert WorkspaceDescriptor stub
						return testErr
					}
					return nil
				},
			})
		wsb.AddProjector(failingProjQName).SetSync(true).Events().Add(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.QNames(cudQName))
	})
	defer tearDown(app)

	// ok to c.sys.CUD
	respData := sendCUD_3Inserts(t, 1, app)
	require.Equal(2, int(respData["CurrentWLogOffset"].(float64))) // 1st is WorkspaceDescriptor stub insert
	require.Equal(istructs.FirstUserRecordID, istructs.RecordID(respData["NewIDs"].(map[string]interface{})["1"].(float64)))
	require.Equal(istructs.FirstUserRecordID+1, istructs.RecordID(respData["NewIDs"].(map[string]interface{})["2"].(float64)))
	require.Equal(istructs.FirstUserRecordID+2, istructs.RecordID(respData["NewIDs"].(map[string]interface{})["3"].(float64)))

	// 2nd c.sys.CUD -> sync projector failure, expect 500 internal server error
	logCap.Reset()
	respData = sendCUD_3Inserts(t, 1, app, http.StatusInternalServerError)
	require.Equal(testErr.Error(), respData["sys.Error"].(map[string]interface{})["Message"].(string))
	logCap.HasLine("stage=sp.triggeredby", "extension="+projExtension, "woffset=", "poffset=", "evqname=")
	logCap.HasLine("stage=sp.error", testErr.Error(), "extension="+projExtension, "woffset=", "poffset=", "evqname=")
	require.Equal(2, strings.Count(logCap.String(), "stage=sp.error"))

	logCap.EventuallyHasLine(
		"stage=cp.partition_recovery",
		"vapp=sys/voedger",
		"extension=sys._Recovery",
		"partition will be restarted",
	)

	// PLog and record is applied but WLog is not written here because sync projector is failed
	// partition is scheduled to be recovered

	// 3rd c.sys.CUD - > recovery procedure must re-apply 2nd event (PLog, records and WLog), then 3rd event is processed ok (sync projectors are ok)
	require.NoError(triggerAndWaitForRecovery(t, app, 1))
	respData = sendCUD_3Inserts(t, 1, app)
	require.Equal(4, int(respData["CurrentWLogOffset"].(float64)))
	require.Equal(istructs.FirstUserRecordID+6, istructs.RecordID(respData["NewIDs"].(map[string]interface{})["1"].(float64)))
	require.Equal(istructs.FirstUserRecordID+7, istructs.RecordID(respData["NewIDs"].(map[string]interface{})["2"].(float64)))
	require.Equal(istructs.FirstUserRecordID+8, istructs.RecordID(respData["NewIDs"].(map[string]interface{})["3"].(float64)))
}

func TestPartitionRecovery(t *testing.T) {
	t.Run("recovery: scn: At least the first two command requests to a partition and workspace return 503 after initial recovery", func(t *testing.T) {
		require := require.New(t)
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		// Given the server has become ready
		app := setUpRecoveryTestApp(t)
		defer tearDown(app)
		partitionKey := partitionManagerKeyForWSID(1)
		workspaceKey := workspaceManagerKey(1)
		gate := app.recoveryTestControl.workspaces.blockNext(workspaceKey)
		release := sync.OnceFunc(func() { close(gate) })
		defer release()

		// When VAOperator submits command requests to a workspace in partition 1 in this order:
		// | request | timing               | retry policy            |
		// | 1       | after readiness      | none                    |
		status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(1, app))
		require.NoError(err)
		// Then the responses are:
		// | request | status                  |
		// | 1       | 503 Service Unavailable |
		require.Equal(http.StatusServiceUnavailable, status)
		require.NoError(app.recoveryTestControl.partitions.wait(app.ctx, partitionKey))
		require.Equal(0, app.recoveryTestControl.workspaces.startCount(workspaceKey))

		// When VAOperator submits command requests to a workspace in partition 1 in this order:
		// | request | timing               | retry policy            |
		// | 2       | after response 1     | none                    |
		status, err = sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(1, app))
		require.NoError(err)
		// Then the responses are:
		// | request | status                  |
		// | 2       | 503 Service Unavailable |
		require.Equal(http.StatusServiceUnavailable, status)
		require.Equal(1, app.recoveryTestControl.workspaces.startCount(workspaceKey))

		release()
		require.NoError(app.recoveryTestControl.workspaces.wait(app.ctx, workspaceKey))
		// When VAOperator submits command requests to a workspace in partition 1 in this order:
		// | request | timing               | retry policy            |
		// | 3       | after response 2     | 503 Service Unavailable |
		// Then the responses are:
		// | request | status                  |
		// | 3       | 200 OK                  |
		sendCUD_3Inserts(t, 1, app)

		logCap.HasLine(
			"stage=cp.partition_recovery.start",
			"vapp=sys/voedger",
			"extension=sys._Recovery",
			"partid=1",
		)
		logCap.HasLine("stage=cp.partition_recovery.complete",
			"vapp=sys/voedger",
			"extension=sys._Recovery",
			"partid=1",
		)
		logCap.HasLine(
			"stage=cp.workspace_recovery.start",
			"vapp=sys/voedger",
			"extension=sys._Recovery",
			"partid=1",
			"wsid=1",
		)
		logCap.HasLine(
			"stage=cp.workspace_recovery.complete",
			"vapp=sys/voedger",
			"extension=sys._Recovery",
			"partid=1",
			"wsid=1",
		)
	})

	t.Run("partition recovery limit returns service unavailable", func(t *testing.T) {
		require := require.New(t)
		recoverCalled := false
		manager := newPartitionManager(nopRecoveryHooks[partitionKey, appPartition](),
			func(context.Context, *cmdWorkpiece) (*appPartition, error) {
				recoverCalled = true
				return &appPartition{}, nil
			})
		manager.partitions.slots = make(chan struct{})
		key := partitionManagerKeyForWSID(1)
		cmd := &cmdWorkpiece{cmdMes: &implICommandMessage{requestCtx: context.Background()}}

		partition, err := manager.getOrStart(context.Background(), key, cmd)

		require.Nil(partition)
		require.ErrorIs(err, coreutils.SysError{
			HTTPStatus: http.StatusServiceUnavailable,
			Message:    "partition 1: recovery concurrency limit is reached",
		})
		require.False(recoverCalled)
		manager.shutdown()
	})

	t.Run("recovery: scn: An unauthenticated command does not trigger partition recovery", func(t *testing.T) {
		require := require.New(t)
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		// Given partition 1 is deployed but unrecovered
		app := setUpRecoveryTestApp(t)
		defer tearDown(app)

		request := newCUD_3Inserts_Request(1, app)
		request.Header = map[string]string{httpu.Authorization: "Bearer invalid-token"}
		// When VAOperator submits a command to workspace 1 with an invalid token without automatic retries
		status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, request)
		require.NoError(err)
		// Then response status is "401 Unauthorized"
		require.Equal(http.StatusUnauthorized, status)
		// And no partition or workspace recovery is started by the command
		require.Equal(0, app.recoveryTestControl.partitions.startCount(partitionManagerKeyForWSID(1)))
		require.Equal(0, app.recoveryTestControl.workspaces.startCount(workspaceManagerKey(1)))
		logCap.NotContains("stage=cp.partition_recovery.start", "stage=cp.workspace_recovery.start")
	})

	t.Run("recovery: scn: A command succeeds in another partition while partition recovery is running", func(t *testing.T) {
		require := require.New(t)
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		// And workspace 2 in partition 0 has only its initialization event at WLog offset 1 and no user record IDs
		app := setUpRecoveryTestApp(t)
		defer tearDown(app)

		recoveringKey := partitionManagerKeyForWSID(1)
		gate := app.recoveryTestControl.partitions.blockNext(recoveringKey)
		release := sync.OnceFunc(func() { close(gate) })
		defer release()
		// Given partition 1 recovery is in progress
		sendCUD_3Inserts_WithSender(t, 1, app, app.rawRequestSender, http.StatusServiceUnavailable)

		// When VAOperator inserts an ordinary record into workspace 2, retrying "503 Service Unavailable" responses
		response := sendCUDWithBody(t, 2, app, `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.TestCDoc"}}]}`)
		// Then the insert returns "200 OK" with CurrentWLogOffset 2 and new ordinary record ID FirstUserRecordID
		requireIssuedNumbers(t, response, 2, istructs.FirstUserRecordID)
		require.Len(response["NewIDs"], 1)
		// And partition 1 recovery has not completed when the insert succeeds
		require.Equal(0, countCapturedLogLines(logCap, "stage=cp.partition_recovery.complete", "partid=1"))
		require.Equal(1, app.recoveryTestControl.partitions.startCount(recoveringKey))

		release()
		require.NoError(app.recoveryTestControl.partitions.wait(app.ctx, recoveringKey))
		sendCUD_3Inserts(t, 1, app)
	})

	t.Run("recovery: scn: Requests to a partition being recovered share one recovery", func(t *testing.T) {
		require := require.New(t)
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		// Given partition 1 is deployed but unrecovered
		// And workspace 1 has only its initialization event at WLog offset 1 and no user record IDs
		app := setUpRecoveryTestApp(t)
		defer tearDown(app)

		key := partitionManagerKeyForWSID(1)
		gate := app.recoveryTestControl.partitions.blockNext(key)
		release := sync.OnceFunc(func() { close(gate) })
		defer release()
		// When VAOperator submits commands inserting three ordinary records into workspace 1 at these times:
		// | request     | timing                                   | retry policy            |
		// | 1           | before partition recovery starts         | none                    |
		// Then the responses are:
		// | request     | status                  |
		// | 1 through 6 | 503 Service Unavailable |
		sendCUD_3Inserts_WithSender(t, 1, app, app.rawRequestSender, http.StatusServiceUnavailable)

		const concurrentRequests = 5
		type result struct {
			status int
			err    error
		}
		results := make(chan result, concurrentRequests)
		// When VAOperator submits commands inserting three ordinary records into workspace 1 at these times:
		// | request     | timing                                   | retry policy            |
		// | 2 through 6 | concurrently while recovery is running   | none                    |
		for range concurrentRequests {
			go func() {
				status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(1, app))
				results <- result{status: status, err: err}
			}()
		}
		for range concurrentRequests {
			result := <-results
			require.NoError(result.err)
			// Then the responses are:
			// | request     | status                  |
			// | 1 through 6 | 503 Service Unavailable |
			require.Equal(http.StatusServiceUnavailable, result.status)
		}
		require.Equal(1, app.recoveryTestControl.partitions.startCount(key))
		// And error stage "cp.error" logs message "partition 1: recovery is in progress"
		logCap.HasLine("level=ERROR", "stage=cp.error", "partition 1: recovery is in progress")

		release()
		require.NoError(app.recoveryTestControl.partitions.wait(app.ctx, key))
		// When VAOperator submits commands inserting three ordinary records into workspace 1 at these times:
		// | request     | timing                                   | retry policy            |
		// | 7           | after partition recovery completes       | 503 Service Unavailable |
		// Then the responses are:
		// | request     | status                  |
		// | 7           | 200 OK                  |
		response := sendCUD_3Inserts(t, 1, app)
		// And response 7 contains CurrentWLogOffset 2 and new record IDs 200001 through 200003
		require.Equal(float64(2), response["CurrentWLogOffset"])
		require.Equal(map[string]interface{}{"1": float64(200001), "2": float64(200002), "3": float64(200003)}, response["NewIDs"])
		// And exactly one partition-recovery start is logged for partition 1
		require.Equal(1, countCapturedLogLines(logCap, "stage=cp.partition_recovery.start", "partid=1"))
	})

	t.Run("recovery: scn: Canceling the initiating request does not stop partition recovery", func(t *testing.T) {
		require := require.New(t)
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		// And workspace 1 has only its initialization event at WLog offset 1 and no user record IDs
		app := setUpRecoveryTestApp(t)
		defer tearDown(app)

		key := partitionManagerKeyForWSID(1)
		gate := app.recoveryTestControl.partitions.blockNext(key)
		release := sync.OnceFunc(func() { close(gate) })
		defer release()
		requestCtx, cancelRequest := context.WithCancel(app.ctx)
		defer cancelRequest()
		// Given partition 1 recovery was started by a command request to workspace 1
		status, err := sendRequestForStatus(requestCtx, app.rawRequestSender, newCUD_3Inserts_Request(1, app))
		require.NoError(err)
		require.Equal(http.StatusServiceUnavailable, status)

		// When VAOperator cancels the initiating request
		cancelRequest()
		release()
		require.NoError(app.recoveryTestControl.partitions.wait(app.ctx, key))
		// Then inserting three ordinary records into workspace 1 after recovery completes, retrying "503 Service Unavailable" responses, returns "200 OK" with CurrentWLogOffset 2 and new record IDs 200001 through 200003
		response := sendCUD_3Inserts(t, 1, app)
		require.Equal(float64(2), response["CurrentWLogOffset"])
		require.Equal(map[string]interface{}{"1": float64(200001), "2": float64(200002), "3": float64(200003)}, response["NewIDs"])
		// And exactly one partition-recovery start is logged for partition 1
		require.Equal(1, countCapturedLogLines(logCap, "stage=cp.partition_recovery.start", "partid=1"))
	})

	t.Run("recovery: scn: The next request reports a failed partition recovery and starts a new attempt", func(t *testing.T) {
		//
		//	request 1 ──> start recovery 1 ──> 503 Service Unavailable
		//	                     |
		//	                     +──> failure retained
		//	                              |
		//	request 2 ─────────────────────+──> 500 Internal Server Error
		//	        +──> start recovery 2 (blocked)
		//	                     |
		//	request 3 ───────────+────────────> 503; no additional recovery
		//	                     |
		//	              unblock retry
		//	                     |
		//	                     v
		//	                  recovered
		//	                     |
		//	request 4 ───────────+────────────> success
		require := require.New(t)
		// Given partition 1 is deployed but unrecovered
		// And workspace 1 has only its initialization event at WLog offset 1 and no user record IDs
		app := setUpRecoveryTestApp(t)
		defer tearDown(app)

		key := partitionManagerKeyForWSID(1)
		recoveryErr := errors.New("PLog read failed")
		// And its first partition recovery fails with "PLog read failed"
		app.recoveryTestControl.partitions.failNext(key, recoveryErr)
		// When VAOperator submits commands inserting three ordinary records into workspace 1 at these times:
		// | request | timing                               | retry policy            |
		// | 1       | before partition recovery starts     | none                    |
		// Then the responses are:
		// | request | status                    |
		// | 1       | 503 Service Unavailable   |
		sendCUD_3Inserts_WithSender(t, 1, app, app.rawRequestSender, http.StatusServiceUnavailable)
		require.ErrorIs(app.recoveryTestControl.partitions.wait(app.ctx, key), recoveryErr)
		require.Equal(1, app.recoveryTestControl.partitions.startCount(key))

		gate := app.recoveryTestControl.partitions.blockNext(key)
		release := sync.OnceFunc(func() { close(gate) })
		defer release()
		// When VAOperator submits commands inserting three ordinary records into workspace 1 at these times:
		// | request | timing                               | retry policy            |
		// | 2       | after the first recovery fails       | none                    |
		// Then the responses are:
		// | request | status                    |
		// | 2       | 500 Internal Server Error |
		respData := sendCUD_3Inserts_WithSender(t, 1, app, app.rawRequestSender, http.StatusInternalServerError)
		sysError := respData["sys.Error"].(map[string]interface{})
		// And request 2 returns error "partition 1: recovery failed: PLog read failed" and starts one new recovery attempt
		require.Equal("partition 1: recovery failed: PLog read failed", sysError["Message"])
		require.Equal(2, app.recoveryTestControl.partitions.startCount(key))

		// When VAOperator submits commands inserting three ordinary records into workspace 1 at these times:
		// | request | timing                               | retry policy            |
		// | 3       | while the new attempt is in progress | none                    |
		// Then the responses are:
		// | request | status                    |
		// | 3       | 503 Service Unavailable   |
		sendCUD_3Inserts_WithSender(t, 1, app, app.rawRequestSender, http.StatusServiceUnavailable)
		// And request 3 does not start another recovery attempt
		require.Equal(2, app.recoveryTestControl.partitions.startCount(key))
		release()
		require.NoError(app.recoveryTestControl.partitions.wait(app.ctx, key))
		// When VAOperator submits commands inserting three ordinary records into workspace 1 at these times:
		// | request | timing                               | retry policy            |
		// | 4       | after the new attempt completes      | 503 Service Unavailable |
		// Then the responses are:
		// | request | status                    |
		// | 4       | 200 OK                    |
		respData = sendCUD_3Inserts(t, 1, app)
		// And the fourth response contains CurrentWLogOffset 2 and new record IDs 200001 through 200003
		require.Equal(float64(2), respData["CurrentWLogOffset"])
		require.Equal(map[string]interface{}{"1": float64(200001), "2": float64(200002), "3": float64(200003)}, respData["NewIDs"])
	})

	t.Run("command processors shuts down -> recovery attempts finished by context cancellation", func(t *testing.T) {
		//
		//	request                 service                  recovery worker
		//	   |                       |                            |
		//	   +── command ──────────> | ── start ──────────-─────> | (blocked)
		//	   | <── 503 ──────────────+                            |
		//	   |                       |                            |
		//	shutdown ────────────────> | ── cancel context ───-───> |
		//	                           |          waits             +── record context.Canceled
		//	                           | <──────────────────────────+── exit
		//	                           +── Run returns              |
		//	                                      recovery starts: 1
		require := require.New(t)
		app := setUpRecoveryTestApp(t)

		key := partitionManagerKeyForWSID(1)
		app.recoveryTestControl.partitions.blockNext(key)
		sendCUD_3Inserts_WithSender(t, 1, app, app.rawRequestSender, http.StatusServiceUnavailable)

		app.n10nBrokerCleanup()
		app.cancel()
		<-app.done
		require.ErrorIs(app.recoveryTestControl.partitions.wait(context.Background(), key), context.Canceled)
		require.Equal(1, app.recoveryTestControl.partitions.startCount(key))
	})
}

func addStandardTestRoles(wsb appdef.IWorkspaceBuilder) {
	wsb.AddRole(iauthnz.QNameRoleAuthenticatedUser)
	wsb.AddRole(iauthnz.QNameRoleEveryone)
	wsb.AddRole(iauthnz.QNameRoleSystem)
}

func addTestCUDCommand(wsb appdef.IWorkspaceBuilder, cfg *istructsmem.AppConfigType) {
	wsb.AddCommand(istructs.QNameCommandCUD)
	cfg.Resources.Add(istructsmem.NewCommandFunction(istructs.QNameCommandCUD, istructsmem.NullCommandExec))
}

func addTestCUDRecords(wsb appdef.IWorkspaceBuilder) {
	wsb.AddCRecord(testCRecord)
	wsb.AddCDoc(testCDoc).AddContainer("TestCRecord", testCRecord, 0, 1)
	wsb.AddWDoc(testWDoc)
}

func setUpRecoveryTestApp(t *testing.T, options ...testAppOption) testApp {
	t.Helper()
	return setUp(t, func(wsb appdef.IWorkspaceBuilder, cfg *istructsmem.AppConfigType) {
		addTestCUDRecords(wsb)
		wsb.AddCDoc(testSingleton).SetSingleton()
		addTestCUDCommand(wsb, cfg)
		wsb.AddObject(testNoCUDParams).AddField("Marker", appdef.DataKind_int32, false)
		wsb.AddCommand(testNoCUDCommand).SetParam(testNoCUDParams)
		addStandardTestRoles(wsb)
		cfg.Resources.Add(istructsmem.NewCommandFunction(testNoCUDCommand, istructsmem.NullCommandExec))
	}, options...)
}

func restartCmdProc(app *testApp) {
	app.cancel()
	<-app.done
	app.ctx, app.cancel = context.WithCancel(context.Background())
	app.done = make(chan struct{})
	go func() {
		app.cmdProcService.Run(app.ctx)
		close(app.done)
	}()
}

//	hook --> injected error -------+--> skip next --> skip cleanup --> engine leaks
//	     \-> blocked gate + cancel -+
//
// Hold all other engines so borrowing again succeeds only if recovery releases
// its engine despite the hook's early return.
func TestPartitionRecoveryHookCleanup(t *testing.T) {
	for _, test := range []struct {
		name       string
		cancelGate bool
	}{
		{name: "injected failure"},
		{name: "canceled gate", cancelGate: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)
			var parts appparts.IAppPartitions
			app := setUpRecoveryTestApp(t, func(options *testAppOptions) {
				options.appPartsForProcessor = func(actual appparts.IAppPartitions) appparts.IAppPartitions {
					parts = actual
					return actual
				}
			})
			defer tearDown(app)
			key := partitionManagerKeyForWSID(1)

			// Hold every engine so only recovery cleanup can make one available.
			var heldPartitions []appparts.IAppPartition
			defer func() {
				for _, held := range heldPartitions {
					held.Release()
				}
			}()
			for range testAppEngines[appparts.ProcessorKind_Command] - 1 {
				borrowed, err := parts.Borrow(key.appQName, key.partitionID, appparts.ProcessorKind_Command)
				require.NoError(err)
				heldPartitions = append(heldPartitions, borrowed)
			}
			borrowed, err := parts.Borrow(key.appQName, key.partitionID, appparts.ProcessorKind_Command)
			require.NoError(err)
			_, err = parts.Borrow(key.appQName, key.partitionID, appparts.ProcessorKind_Command)
			require.ErrorIs(err, appparts.ErrNotAvailableEngines)

			control := newRecoveryTestAttempts[partitionKey, appPartition]()
			expectedErr := errors.New("injected recovery failure")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if test.cancelGate {
				control.blockNext(key)
				expectedErr = context.Canceled
			} else {
				control.failNext(key, expectedErr)
			}
			manager := newPartitionManager(control.hooks(), func(context.Context, *cmdWorkpiece) (*appPartition, error) {
				return nil, errors.New("production recovery must be skipped")
			})
			defer func() {
				cancel()
				manager.shutdown()
			}()
			cmd := &cmdWorkpiece{
				appPart: borrowed,
				cmdMes: &implICommandMessage{
					appQName: key.appQName, partitionID: key.partitionID, requestCtx: t.Context(),
				},
			}
			defer cmd.Release()
			_, err = manager.getOrStart(ctx, key, cmd)
			require.ErrorContains(err, errRecoveryInProgress.Error())
			require.Nil(cmd.appPart, "the admitted recovery owns the borrowed partition")
			cmd.Release()
			if test.cancelGate {
				cancel()
			}
			require.ErrorIs(control.wait(t.Context(), key), expectedErr)

			returned, err := parts.Borrow(key.appQName, key.partitionID, appparts.ProcessorKind_Command)
			require.NoError(err, "recovery must return its borrowed engine even when the hook skips recovery")
			returned.Release()
		})
	}
}

func TestRecoveryCheckpoints_RecordIDsRecovery(t *testing.T) {

	t.Run("read out from checkpoint: checkpoints are on final events", func(t *testing.T) {
		storage := testutils.NewCheckpointStorageMem()
		app := setUpRecoveryTestApp(t, withCheckpointStorage(storage))
		defer func() { tearDown(app) }()

		previous := sendCUD_3Inserts(t, 1, app)
		lastWLogOffset := istructs.Offset(previous["CurrentWLogOffset"].(float64))
		key := partitionManagerKeyForWSID(1)
		savedPLogOffset := nextPLogOffset(t, app.appStructs, key.partitionID) - 1
		savedWLogOffset := lastWLogOffset
		testRecoveryFromCheckpoints(t, &app, storage, savedPLogOffset, savedWLogOffset, lastWLogOffset+1,
			istructs.RecordID(previous["NewIDs"].(map[string]interface{})["3"].(float64))+1)
	})

	t.Run("read out from checkpoint: ID-bearing events after the WLog checkpoint", func(t *testing.T) {
		storage := testutils.NewCheckpointStorageMem()
		app := setUpRecoveryTestApp(t, withCheckpointStorage(storage))
		defer func() { tearDown(app) }()

		// Create newer events beyond the checkpoints to exercise log tail scans.
		first := sendCUD_3Inserts(t, 1, app)
		sendCUD_3Inserts(t, 1, app)
		previous := sendCUD_3Inserts(t, 1, app)
		lastWLogOffset := istructs.Offset(previous["CurrentWLogOffset"].(float64))
		key := partitionManagerKeyForWSID(1)
		savedPLogOffset := nextPLogOffset(t, app.appStructs, key.partitionID) - 2
		savedWLogOffset := istructs.Offset(first["CurrentWLogOffset"].(float64))
		testRecoveryFromCheckpoints(t, &app, storage, savedPLogOffset, savedWLogOffset, lastWLogOffset+1,
			istructs.RecordID(previous["NewIDs"].(map[string]interface{})["3"].(float64))+1)
	})

	t.Run("read out from checkpoint: non-ID-bearing events after the WLog checkpoint", func(t *testing.T) {
		storage := testutils.NewCheckpointStorageMem()
		app := setUpRecoveryTestApp(t, withCheckpointStorage(storage))
		defer func() { tearDown(app) }()

		previous := sendCUD_3Inserts(t, 1, app)
		savedWLogOffset := istructs.Offset(previous["CurrentWLogOffset"].(float64))
		// These events advance the log without allocating ordinary record IDs.
		sendCUDWithBody(t, 1, app, `{"cuds":[
			{"fields":{"sys.ID":1,"sys.QName":"test.TestSingleton"}}
		]}`)
		lastEvent := sendNoCUDCommand(t, 1, app)
		lastWLogOffset := istructs.Offset(lastEvent["CurrentWLogOffset"].(float64))
		key := partitionManagerKeyForWSID(1)
		savedPLogOffset := nextPLogOffset(t, app.appStructs, key.partitionID) - 1
		testRecoveryFromCheckpoints(t, &app, storage, savedPLogOffset, savedWLogOffset, lastWLogOffset+1,
			istructs.RecordID(previous["NewIDs"].(map[string]interface{})["3"].(float64))+1)
	})

	t.Run("read out from first: missing and zero checkpoints start at the first offset", func(t *testing.T) {
		for _, checkpointState := range []string{"missing", "zero"} {
			t.Run(checkpointState, func(t *testing.T) {
				require := require.New(t)
				logCap := logger.StartCapture(t, logger.LogLevelVerbose)
				storage := testutils.NewCheckpointStorageMem()
				app := setUpRecoveryTestApp(t, withCheckpointStorage(storage))
				defer tearDown(app)

				partitionManagerKey := partitionManagerKeyForWSID(1)
				workspaceManagerKey := workspaceManagerKey(1)
				if checkpointState == "zero" {
					storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), partitionManagerKey.partitionID, checkpoints.PartitionCheckpoint{})
					storage.ForcePutWorkspaceCheckpoint(t, app.appStructs.ClusterAppID(), 1, checkpoints.WorkspaceCheckpoint{})
				}
				app.recoveryTestControl.pLogReads.reset(partitionManagerKey)
				app.recoveryTestControl.wLogReads.reset(workspaceManagerKey)
				logCap.Reset()

				response := sendCUD_3Inserts(t, 1, app)
				reads := app.recoveryTestControl.pLogReads.values(partitionManagerKey)
				require.Len(reads, 1)
				require.Equal(istructs.FirstOffset, reads[0].offset)
				require.Equal(istructs.ReadToTheEnd, reads[0].count)
				require.Equal([]logReadFromTo{{offset: istructs.FirstOffset, count: istructs.ReadToTheEnd}},
					app.recoveryTestControl.wLogReads.values(workspaceManagerKey))
				logCap.NotContains("cp.partition_recovery.checkpoint.invalid",
					"cp.workspace_recovery.checkpoint.invalid")
				require.Equal(2, int(response["CurrentWLogOffset"].(float64)))
				require.Equal(istructs.FirstUserRecordID,
					istructs.RecordID(response["NewIDs"].(map[string]interface{})["1"].(float64)))
			})
		}
	})

	t.Run("read out from checkpoint: no ID-bearing events at all", func(t *testing.T) {
		storage := testutils.NewCheckpointStorageMem()
		app := setUpRecoveryTestApp(t, withCheckpointStorage(storage))
		defer func() { tearDown(app) }()

		// insert a singleton + 12 non-ID-beraing events
		lastEvent := sendCUDWithBody(t, 1, app, `{"cuds":[
			{"fields":{"sys.ID":1,"sys.QName":"test.TestSingleton"}}
		]}`)
		for range 12 {
			lastEvent = sendNoCUDCommand(t, 1, app)
		}
		lastWLogOffset := istructs.Offset(lastEvent["CurrentWLogOffset"].(float64))

		key := partitionManagerKeyForWSID(1)
		savedPLogOffset := nextPLogOffset(t, app.appStructs, key.partitionID) - 1
		testRecoveryFromCheckpoints(t, &app, storage, savedPLogOffset, 0, lastWLogOffset+1,
			istructs.FirstUserRecordID)
	})

	t.Run("read out from first: partition checkpoint beyond the log end", func(t *testing.T) {
		storage := testutils.NewCheckpointStorageMem()
		app := setUpRecoveryTestApp(t, withCheckpointStorage(storage))
		defer func() { tearDown(app) }()

		previous := sendCUD_3Inserts(t, 1, app)
		key := partitionManagerKeyForWSID(1)
		malformedOffset := nextPLogOffset(t, app.appStructs, key.partitionID) + 10
		testRecoveryFromInvalidPartitionCheckpoint(t, &app, storage, malformedOffset,
			istructs.Offset(previous["CurrentWLogOffset"].(float64))+1,
			istructs.RecordID(previous["NewIDs"].(map[string]interface{})["3"].(float64))+1)
	})

	t.Run("read out from first: workspace checkpoint beyond the log end", func(t *testing.T) {
		storage := testutils.NewCheckpointStorageMem()
		app := setUpRecoveryTestApp(t, withCheckpointStorage(storage))
		defer func() { tearDown(app) }()

		previous := sendCUD_3Inserts(t, 1, app)
		key := partitionManagerKeyForWSID(1)
		malformedOffset := istructs.Offset(previous["CurrentWLogOffset"].(float64)) + 10
		savedPLogOffset := nextPLogOffset(t, app.appStructs, key.partitionID) - 1
		testRecoveryFromInvalidWorkspaceCheckpoint(t, &app, storage, savedPLogOffset, malformedOffset,
			istructs.Offset(previous["CurrentWLogOffset"].(float64))+1,
			istructs.RecordID(previous["NewIDs"].(map[string]interface{})["3"].(float64))+1)
	})

	t.Run("read out from first: partition checkpoint event is missing but later events exist", func(t *testing.T) {
		require := require.New(t)
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		storage := testutils.NewCheckpointStorageMem()
		app := setUpRecoveryTestApp(t, withCheckpointStorage(storage))
		defer tearDown(app)

		for range 3 {
			sendCUD_3Inserts(t, 1, app)
		}
		key := partitionManagerKeyForWSID(1)
		nextPLogOffset := nextPLogOffset(t, app.appStructs, key.partitionID)
		missingOffset := nextPLogOffset - 2
		storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), key.partitionID,
			checkpoints.PartitionCheckpoint{LastPLogOffset: missingOffset})
		events := &pLogCheckpointSkippingEvents{
			IEvents:          app.appStructs.Events(),
			partitionID:      key.partitionID,
			checkpointOffset: missingOffset,
		}
		appStructs := &appStructsWithTestEvents{IAppStructs: app.appStructs, events: events}
		control := newRecoveryTestControl()
		cmdProc := newDirectRecoveryCmdProc(storage, control)
		cmd := &cmdWorkpiece{
			cmdMes: &implICommandMessage{
				requestCtx:  t.Context(),
				appQName:    testAppName,
				partitionID: key.partitionID,
			},
			appStructs: appStructs,
		}
		control.pLogReads.reset(key)
		logCap.Reset()

		recovered, err := cmdProc.recoverPartition(t.Context(), cmd)
		require.NoError(err)
		require.Equal(nextPLogOffset, recovered.nextPLogOffset)
		requireCheckpointFallbackReads(t, missingOffset, control.pLogReads.values(key))
		logCap.HasLine("level=ERROR", "stage=cp.partition_recovery.checkpoint.invalid",
			fmt.Sprintf("stored PLog offset %d does not identify an existing event", missingOffset))
	})

	t.Run("read out from first: workspace checkpoint event is missing but later events exist", func(t *testing.T) {
		require := require.New(t)
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		storage := testutils.NewCheckpointStorageMem()
		app := setUpRecoveryTestApp(t, withCheckpointStorage(storage))
		defer tearDown(app)

		first := sendCUD_3Inserts(t, 1, app)
		last := sendCUD_3Inserts(t, 1, app)
		key := workspaceManagerKey(1)
		missingOffset := istructs.Offset(first["CurrentWLogOffset"].(float64))
		storage.ForcePutWorkspaceCheckpoint(t, app.appStructs.ClusterAppID(), key.wsid,
			checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: missingOffset})
		events := &wLogCheckpointSkippingEvents{
			IEvents:          app.appStructs.Events(),
			wsid:             key.wsid,
			checkpointOffset: missingOffset,
		}
		appStructs := &appStructsWithTestEvents{IAppStructs: app.appStructs, events: events}
		control := newRecoveryTestControl()
		cmdProc := newDirectRecoveryCmdProc(storage, control)
		control.wLogReads.reset(key)
		logCap.Reset()

		recovered, err := cmdProc.recoverWorkspace(t.Context(), t.Context(), key, appStructs)
		require.NoError(err)
		require.Equal(istructs.Offset(last["CurrentWLogOffset"].(float64))+1, recovered.NextWLogOffset)
		nextRecordID, err := recovered.idGenerator.NextID(1)
		require.NoError(err)
		require.Equal(
			istructs.RecordID(last["NewIDs"].(map[string]interface{})["3"].(float64))+1,
			nextRecordID,
		)
		requireCheckpointFallbackReads(t, missingOffset, control.wLogReads.values(key))
		logCap.HasLine("level=ERROR", "stage=cp.workspace_recovery.checkpoint.invalid",
			fmt.Sprintf("stored WLog offset %d does not identify an ID-bearing event", missingOffset))
	})

	t.Run("read out from first: workspace checkpoint points to a non-ID-bearing event", func(t *testing.T) {
		storage := testutils.NewCheckpointStorageMem()
		app := setUpRecoveryTestApp(t, withCheckpointStorage(storage))
		defer func() { tearDown(app) }()

		idBearingEvent := sendCUD_3Inserts(t, 1, app)
		nonIDBearingEvent := sendCUDWithBody(t, 1, app, `{"cuds":[
			{"fields":{"sys.ID":1,"sys.QName":"test.TestSingleton"}}
		]}`)
		lastEvent := sendNoCUDCommand(t, 1, app)
		key := partitionManagerKeyForWSID(1)
		checkpointOffset := istructs.Offset(nonIDBearingEvent["CurrentWLogOffset"].(float64))
		savedPLogOffset := nextPLogOffset(t, app.appStructs, key.partitionID) - 1
		testRecoveryFromInvalidWorkspaceCheckpoint(t, &app, storage, savedPLogOffset, checkpointOffset,
			istructs.Offset(lastEvent["CurrentWLogOffset"].(float64))+1,
			istructs.RecordID(idBearingEvent["NewIDs"].(map[string]interface{})["3"].(float64))+1)
	})
}

func TestSequenceAllocation(t *testing.T) {
	t.Run("sequences: scn: The first insert in a new workspace uses FirstOffset and FirstUserRecordID", func(t *testing.T) {
		require := require.New(t)
		// Given a new workspace has an empty WLog
		app, _, _ := newEmptyRecoveryCommandApp(t)
		// When VAOperator submits an ordinary record insert into the new workspace, retrying "503 Service Unavailable" responses
		response := sendCUDWithBody(t, emptyRecoveryWSID, *app, emptyRecoveryInsert)
		// Then the insert returns "200 OK" with CurrentWLogOffset FirstOffset and new ordinary record ID FirstUserRecordID
		requireIssuedNumbers(t, response, istructs.FirstOffset, istructs.FirstUserRecordID)
		require.Len(response["NewIDs"], 1)
	})
}

func TestRecoveryFeatureCommandProcessorScenarios(t *testing.T) {
	t.Run("recovery: scn: Empty PLog recovery preserves initial sequence numbers: absent", func(t *testing.T) {
		require := require.New(t)
		// | checkpoint       |
		// | absent           |
		// Given the PLog is empty
		app, storage, logs := newEmptyRecoveryCommandApp(t)
		// And the stored partition checkpoint is <checkpoint>
		// checkpoint = absent
		_, exists, err := storage.GetPartitionCheckpoint(app.appStructs.ClusterAppID(), 0)
		require.NoError(err)
		require.False(exists)
		_, exists, err = storage.GetWorkspaceCheckpoint(app.appStructs.ClusterAppID(), emptyRecoveryWSID)
		require.NoError(err)
		require.False(exists)
		// When VAOperator restarts the VVM
		restartCmdProc(app)
		logs.Reset()
		// Then inserting an ordinary record into a workspace, retrying "503 Service Unavailable" responses, returns "200 OK" with CurrentWLogOffset 1 and new ordinary record ID 200001
		response := sendCUDWithBody(t, emptyRecoveryWSID, *app, emptyRecoveryInsert)
		requireIssuedNumbers(t, response, istructs.FirstOffset, istructs.FirstUserRecordID)
		require.Len(response["NewIDs"], 1)
		logs.NotContains("cp.partition_recovery.checkpoint.invalid", "cp.workspace_recovery.checkpoint.invalid")
	})

	t.Run("recovery: scn: Empty PLog recovery preserves initial sequence numbers: lastPLogOffset 0", func(t *testing.T) {
		require := require.New(t)
		// | checkpoint       |
		// | lastPLogOffset 0 |
		// Given the PLog is empty
		app, storage, logs := newEmptyRecoveryCommandApp(t)
		// And the stored partition checkpoint is <checkpoint>
		// checkpoint = lastPLogOffset 0
		storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), 0,
			checkpoints.PartitionCheckpoint{LastPLogOffset: 0})
		// When VAOperator restarts the VVM
		restartCmdProc(app)
		logs.Reset()
		// Then inserting an ordinary record into a workspace, retrying "503 Service Unavailable" responses, returns "200 OK" with CurrentWLogOffset 1 and new ordinary record ID 200001
		response := sendCUDWithBody(t, emptyRecoveryWSID, *app, emptyRecoveryInsert)
		requireIssuedNumbers(t, response, istructs.FirstOffset, istructs.FirstUserRecordID)
		require.Len(response["NewIDs"], 1)
		logs.NotContains("cp.partition_recovery.checkpoint.invalid")
	})

	t.Run("recovery: scn: Empty PLog recovery preserves initial sequence numbers: lastPLogOffset 5", func(t *testing.T) {
		require := require.New(t)
		// | checkpoint       |
		// | lastPLogOffset 5 |
		// Given the PLog is empty
		app, storage, logs := newEmptyRecoveryCommandApp(t)
		// And the stored partition checkpoint is <checkpoint>
		// checkpoint = lastPLogOffset 5
		storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), 0,
			checkpoints.PartitionCheckpoint{LastPLogOffset: 5})
		// When VAOperator restarts the VVM
		restartCmdProc(app)
		logs.Reset()
		// Then inserting an ordinary record into a workspace, retrying "503 Service Unavailable" responses, returns "200 OK" with CurrentWLogOffset 1 and new ordinary record ID 200001
		response := sendCUDWithBody(t, emptyRecoveryWSID, *app, emptyRecoveryInsert)
		requireIssuedNumbers(t, response, istructs.FirstOffset, istructs.FirstUserRecordID)
		require.Len(response["NewIDs"], 1)
		requireLogError(logs, "cp.partition_recovery.checkpoint.invalid",
			"stored PLog offset 5 does not identify an existing event; retrying from FirstOffset")
	})

	t.Run("recovery: scn: Invalid workspace checkpoint over an empty WLog", func(t *testing.T) {
		require := require.New(t)
		// Given the WLog is empty
		app, storage, logs := newEmptyRecoveryCommandApp(t)
		// And the stored workspace checkpoint is wlogIDBearingOffset 5
		storage.ForcePutWorkspaceCheckpoint(t, app.appStructs.ClusterAppID(), emptyRecoveryWSID,
			checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: 5})
		// And the VVM has restarted and completed partition recovery
		restartCmdProc(app)
		status, recoveryErr := sendRequestForStatus(app.ctx, app.rawRequestSender, emptyRecoveryRequest(*app))
		require.NoError(recoveryErr)
		require.Equal(http.StatusServiceUnavailable, status)
		require.NoError(app.recoveryTestControl.partitions.wait(app.ctx, recoveryKeyForTestApp(*app, emptyRecoveryWSID)))
		logs.Reset()
		// When VAOperator submits an ordinary record insert into the unrecovered workspace, retrying "503 Service Unavailable" responses
		response := sendCUDWithBody(t, emptyRecoveryWSID, *app, emptyRecoveryInsert)
		// Then the insert returns "200 OK" with CurrentWLogOffset FirstOffset and new ordinary record ID FirstUserRecordID
		requireIssuedNumbers(t, response, istructs.FirstOffset, istructs.FirstUserRecordID)
		require.Len(response["NewIDs"], 1)
		// And error stage "cp.workspace_recovery.checkpoint.invalid" is logged
		// And error message is "stored WLog offset 5 does not identify an ID-bearing event; retrying from FirstOffset"
		requireLogError(logs, "cp.workspace_recovery.checkpoint.invalid",
			"stored WLog offset 5 does not identify an ID-bearing event; retrying from FirstOffset")
	})
}

const (
	emptyRecoveryWSID   istructs.WSID = 100
	emptyRecoveryInsert               = `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.TestCDoc"}}]}`
)

func emptyRecoveryRequest(app testApp) bus.Request {
	request := newCUD_3Inserts_Request(emptyRecoveryWSID, app)
	request.Body = []byte(emptyRecoveryInsert)
	return request
}

func newEmptyRecoveryCommandApp(t *testing.T) (*testApp, *testutils.CheckpointStorage, logger.ILogCaptor) {
	t.Helper()
	require := require.New(t)
	logs := logger.StartCapture(t, logger.LogLevelVerbose)
	storage := testutils.NewCheckpointStorageMem()
	app := setUpRecoveryTestApp(t, withCheckpointStorage(storage), withTestWorkspaces(), withTestAppPartitions(1))
	t.Cleanup(func() { tearDown(app) })
	id, err := app.appStructs.Records().GetSingletonID(appdef.QNameCDocWorkspaceDescriptor)
	require.NoError(err)
	// Seed only system metadata so the first command starts with genuinely empty logs.
	require.NoError(app.appStructs.Records().PutJSON(emptyRecoveryWSID, map[appdef.FieldName]any{
		appdef.SystemField_QName: appdef.QNameCDocWorkspaceDescriptor.String(),
		appdef.SystemField_ID:    id,
		"WSKind":                 appdef.NewQName(appdef.SysPackage, "TestWSKind").String(), "Status": int32(0), "InitCompletedAtMs": int64(1),
	}))
	events := 0
	require.NoError(app.appStructs.Events().ReadPLog(t.Context(), 0, istructs.FirstOffset, istructs.ReadToTheEnd,
		func(_ istructs.Offset, event istructs.IPLogEvent) error {
			event.Release()
			events++
			return nil
		}))
	require.NoError(app.appStructs.Events().ReadWLog(t.Context(), emptyRecoveryWSID, istructs.FirstOffset, istructs.ReadToTheEnd,
		func(_ istructs.Offset, event istructs.IWLogEvent) error {
			event.Release()
			events++
			return nil
		}))
	require.Zero(events)
	return &app, storage, logs
}

func TestWorkspaceRecovery(t *testing.T) {
	t.Run("recovery: scn: A new request to a workspace being recovered does not start another recovery", func(t *testing.T) {
		require := require.New(t)
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		// Given workspace 42 is unrecovered after partition recovery
		// And its WLog ends at offset 14 with last user record ID 200005
		app := setRecoveryAdmissionApp(t, 42)
		defer tearDown(app)
		key := recoveryWorkspaceKeyForTestApp(app, 42)
		initialStarts := app.recoveryTestControl.workspaces.startCount(key)
		gate := app.recoveryTestControl.workspaces.blockNext(key)
		release := sync.OnceFunc(func() { close(gate) })
		defer release()
		request := newCUD_3Inserts_Request(42, app)
		request.Body = []byte(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.TestCDoc"}}]}`)
		logCap.Reset()

		// When VAOperator submits ordinary record inserts into workspace 42 without automatic retries at these times:
		// | request | timing                                   |
		// | 1       | before workspace recovery starts         |
		status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, request)
		require.NoError(err)
		// Then the responses are:
		// | request | status                  |
		// | 1       | 503 Service Unavailable |
		require.Equal(http.StatusServiceUnavailable, status)
		require.Equal(initialStarts+1, app.recoveryTestControl.workspaces.startCount(key))

		// When VAOperator submits ordinary record inserts into workspace 42 without automatic retries at these times:
		// | request | timing                                   |
		// | 2       | while that recovery is still in progress |
		status, err = sendRequestForStatus(app.ctx, app.rawRequestSender, request)
		require.NoError(err)
		// Then the responses are:
		// | request | status                  |
		// | 2       | 503 Service Unavailable |
		require.Equal(http.StatusServiceUnavailable, status)
		require.Equal(initialStarts+1, app.recoveryTestControl.workspaces.startCount(key))
		// And error stage "cp.error" logs message "workspace 42: recovery is in progress"
		logCap.HasLine("stage=cp.error", "workspace 42: recovery is in progress")

		release()
		require.NoError(app.recoveryTestControl.workspaces.wait(app.ctx, key))
		// When VAOperator submits ordinary record inserts into workspace 42 without automatic retries at these times:
		// | request | timing                                   |
		// | 3       | after that recovery completes            |
		// Then the responses are:
		// | request | status                  |
		// | 3       | 200 OK                  |
		response := sendRequestWithSender(t, app, app.rawRequestSender, request)
		// And the third response contains CurrentWLogOffset 15 and new ordinary record ID 200006
		requireIssuedNumbers(t, response, 15, 200006)
		require.Len(response["NewIDs"], 1)
		require.Equal(initialStarts+1, app.recoveryTestControl.workspaces.startCount(key))
		// And exactly one workspace-recovery start is logged for workspace 42
		require.Equal(1, countCapturedLogLines(logCap,
			"stage=cp.workspace_recovery.start", "wsid=42"))
	})

	t.Run("recovery: scn: The next request reports a failed workspace recovery and starts a new attempt", func(t *testing.T) {
		require := require.New(t)
		// Given workspace 42 is unrecovered after partition recovery
		// And its WLog ends at offset 14 with last user record ID 200005
		app := setRecoveryAdmissionApp(t, 42)
		defer tearDown(app)
		key := recoveryWorkspaceKeyForTestApp(app, 42)
		initialStarts := app.recoveryTestControl.workspaces.startCount(key)
		injectedErr := errors.New("WLog read failed")
		// And its first workspace recovery fails with "WLog read failed"
		app.recoveryTestControl.workspaces.failNext(key, injectedErr)

		// When VAOperator submits inserts into workspace 42 at these times:
		// | request | timing                               |
		// | 1       | before workspace recovery starts     |
		status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(42, app))
		require.NoError(err)
		// Then the responses are:
		// | request | status                    |
		// | 1       | 503 Service Unavailable   |
		require.Equal(http.StatusServiceUnavailable, status)
		require.ErrorIs(app.recoveryTestControl.workspaces.wait(app.ctx, key), injectedErr)

		retryGate := app.recoveryTestControl.workspaces.blockNext(key)
		// When VAOperator submits inserts into workspace 42 at these times:
		// | request | timing                               |
		// | 2       | after the first recovery fails       |
		// Then the responses are:
		// | request | status                    |
		// | 2       | 500 Internal Server Error |
		response := sendCUD_3Inserts_WithSender(t, 42, app, app.rawRequestSender, http.StatusInternalServerError)
		// And request 2 returns error "workspace 42: recovery failed: WLog read failed" and starts one new recovery attempt
		require.Equal("workspace 42: recovery failed: WLog read failed",
			response["sys.Error"].(map[string]interface{})["Message"])
		require.Equal(initialStarts+2, app.recoveryTestControl.workspaces.startCount(key))
		// When VAOperator submits inserts into workspace 42 at these times:
		// | request | timing                               |
		// | 3       | while the new attempt is in progress |
		status, err = sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(42, app))
		require.NoError(err)
		// Then the responses are:
		// | request | status                    |
		// | 3       | 503 Service Unavailable   |
		require.Equal(http.StatusServiceUnavailable, status)
		// And request 3 does not start another recovery attempt
		require.Equal(initialStarts+2, app.recoveryTestControl.workspaces.startCount(key))

		close(retryGate)
		require.NoError(app.recoveryTestControl.workspaces.wait(app.ctx, key))
		// When VAOperator submits inserts into workspace 42 at these times:
		// | request | timing                               |
		// | 4       | after the new attempt completes      |
		// Then the responses are:
		// | request | status                    |
		// | 4       | 200 OK                    |
		response = sendCUDWithBody(t, 42, app,
			`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.TestCDoc"}}]}`)
		// And the fourth response contains CurrentWLogOffset 15 and new ordinary record ID 200006
		requireIssuedNumbers(t, response, 15, 200006)
	})

	t.Run("recovery: scn: Recovery limit rejects without queueing and permits a later retry", func(t *testing.T) {
		require := require.New(t)
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)
		// Given the workspace recovery limit is 4
		require.Equal(uint(4), numWSRecoverers)
		// And workspaces 42 and 43 are unrecovered after partition recovery
		// And both WLogs end at offset 14 with last user record ID 200005
		app := setRecoveryAdmissionApp(t, 42, 43, 44, 45, 46)
		defer tearDown(app)
		firstKey := recoveryWorkspaceKeyForTestApp(app, 42)
		secondKey := recoveryWorkspaceKeyForTestApp(app, 43)
		firstInitialStarts := app.recoveryTestControl.workspaces.startCount(firstKey)
		secondInitialStarts := app.recoveryTestControl.workspaces.startCount(secondKey)
		firstGate := app.recoveryTestControl.workspaces.blockNext(firstKey)
		backgroundGates := make(map[workspaceKey]chan struct{})
		defer func() {
			for key, gate := range backgroundGates {
				close(gate)
				waitCtx, cancel := context.WithTimeout(app.ctx, 5*time.Second)
				err := app.recoveryTestControl.workspaces.wait(waitCtx, key)
				cancel()
				require.NoError(err)
			}
		}()
		// And workspace recoveries for workspaces 44 through 46 are running
		for _, wsid := range []istructs.WSID{44, 45, 46} {
			key := recoveryWorkspaceKeyForTestApp(app, wsid)
			backgroundGates[key] = app.recoveryTestControl.workspaces.blockNext(key)
			status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(wsid, app))
			require.NoError(err)
			require.Equal(http.StatusServiceUnavailable, status)
		}
		secondGate := app.recoveryTestControl.workspaces.blockNext(secondKey)
		logCap.Reset()

		// When VAOperator submits inserts in this order:
		// | request | workspace | timing                                 |
		// | 1       | 42        | while workspaces 44 through 46 recover |
		status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(42, app))
		require.NoError(err)
		// Then the responses are:
		// | request | status                  |
		// | 1       | 503 Service Unavailable |
		require.Equal(http.StatusServiceUnavailable, status)
		require.Equal(firstInitialStarts+1, app.recoveryTestControl.workspaces.startCount(firstKey))
		// When VAOperator submits inserts in this order:
		// | request | workspace | timing                                 |
		// | 2       | 43        | before workspace 42 recovery completes |
		status, err = sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(43, app))
		require.NoError(err)
		// Then the responses are:
		// | request | status                  |
		// | 2       | 503 Service Unavailable |
		require.Equal(http.StatusServiceUnavailable, status)
		// And request 2 logs message "workspace 43: recovery concurrency limit is reached" at error stage "cp.error"
		logCap.HasLine("stage=cp.error", "workspace 43: recovery concurrency limit is reached")
		// And no workspace-recovery start is logged for workspace 43 before request 3
		require.Zero(countCapturedLogLines(logCap,
			"stage=cp.workspace_recovery.start", "wsid=43"))

		close(firstGate)
		require.NoError(app.recoveryTestControl.workspaces.wait(app.ctx, firstKey))
		require.Equal(secondInitialStarts, app.recoveryTestControl.workspaces.startCount(secondKey), "rejected recovery was queued")
		// When VAOperator submits inserts in this order:
		// | request | workspace | timing                                 |
		// | 3       | 43        | after workspace 42 recovery completes  |
		status, err = sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(43, app))
		require.NoError(err)
		// Then the responses are:
		// | request | status                  |
		// | 3       | 503 Service Unavailable |
		require.Equal(http.StatusServiceUnavailable, status)
		require.Equal(secondInitialStarts+1, app.recoveryTestControl.workspaces.startCount(secondKey))
		close(secondGate)
		require.NoError(app.recoveryTestControl.workspaces.wait(app.ctx, secondKey))
		require.Equal(1, countCapturedLogLines(logCap,
			"stage=cp.workspace_recovery.start", "wsid=43"))
		// When VAOperator submits inserts in this order:
		// | request | workspace | timing                                 |
		// | 4       | 43        | after workspace 43 recovery completes  |
		// Then the responses are:
		// | request | status                  |
		// | 4       | 200 OK                  |
		response := sendCUDWithBody(t, 43, app,
			`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.TestCDoc"}}]}`)
		// And the fourth response contains CurrentWLogOffset 15 and new ordinary record ID 200006
		requireIssuedNumbers(t, response, 15, 200006)
	})

	t.Run("recovery: scn: A partition reset does not stop ongoing workspace recoveries", func(t *testing.T) {
		require := require.New(t)
		var failNextProjector atomic.Bool
		projectorErr := errors.New("reset partition while workspace recoveries are running")
		app := setUp(t, func(wsb appdef.IWorkspaceBuilder, cfg *istructsmem.AppConfigType) {
			addTestCUDRecords(wsb)
			addTestCUDCommand(wsb, cfg)
			addStandardTestRoles(wsb)
			projectorName := appdef.NewQName("test", "FailNextProjector")
			cfg.AddSyncProjectors(istructs.Projector{
				Name: projectorName,
				Func: func(istructs.IPLogEvent, istructs.IState, istructs.IIntents) error {
					if failNextProjector.Swap(false) {
						return projectorErr
					}
					return nil
				},
			})
			wsb.AddProjector(projectorName).SetSync(true).Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute}, filter.QNames(istructs.QNameCommandCUD))
		}, withTestWorkspaces(1, 3, 5, 7, 9, 11))
		defer tearDown(app)

		// Given the workspace recovery limit is 4
		require.Equal(uint(4), numWSRecoverers)
		// And workspace 1 is recovered
		sendCUD_3Inserts(t, 1, app)

		oldKeys := make([]workspaceKey, 0, numWSRecoverers)
		oldReleases := make([]func(), 0, numWSRecoverers)
		// And workspace recoveries for workspaces 3, 5, 7, and 9 are running in the same partition
		for _, wsid := range []istructs.WSID{3, 5, 7, 9} {
			key := recoveryWorkspaceKeyForTestApp(app, wsid)
			gate := app.recoveryTestControl.workspaces.blockNext(key)
			release := sync.OnceFunc(func() { close(gate) })
			t.Cleanup(release)
			oldKeys = append(oldKeys, key)
			oldReleases = append(oldReleases, release)
			status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(wsid, app))
			require.NoError(err)
			require.Equal(http.StatusServiceUnavailable, status)
			require.Equal(1, app.recoveryTestControl.workspaces.startCount(key))
		}

		// When a command failure resets and recovers the partition
		failNextProjector.Store(true)
		response := sendCUD_3Inserts(t, 1, app, http.StatusInternalServerError)
		require.Equal(projectorErr.Error(), response["sys.Error"].(map[string]interface{})["Message"])
		require.NoError(triggerAndWaitForRecovery(t, app, 1))

		targetKey := recoveryWorkspaceKeyForTestApp(app, 11)
		targetGate := app.recoveryTestControl.workspaces.blockNext(targetKey)
		releaseTarget := sync.OnceFunc(func() { close(targetGate) })
		t.Cleanup(releaseTarget)
		// And VAOperator submits inserts into unrecovered workspace 11 at these times:
		// | request | timing                                          |
		// | 1       | before any detached workspace recovery finishes |
		status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(11, app))
		require.NoError(err)
		// Then the responses are:
		// | request | status                  | reason                        |
		// | 1       | 503 Service Unavailable | recovery limit reached        |
		require.Equal(http.StatusServiceUnavailable, status)
		// And no recovery starts for workspace 11 before request 2
		require.Zero(app.recoveryTestControl.workspaces.startCount(targetKey))

		oldReleases[0]()
		require.NoError(app.recoveryTestControl.workspaces.wait(app.ctx, oldKeys[0]))
		// And VAOperator submits inserts into unrecovered workspace 11 at these times:
		// | request | timing                                          |
		// | 2       | after one detached workspace recovery finishes  |
		status, err = sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(11, app))
		require.NoError(err)
		// Then the responses are:
		// | request | status                  | reason                        |
		// | 2       | 503 Service Unavailable | workspace 11 recovery running |
		require.Equal(http.StatusServiceUnavailable, status)
		// And request 2 starts exactly one recovery for workspace 11
		require.Equal(1, app.recoveryTestControl.workspaces.startCount(targetKey))
		releaseTarget()
		require.NoError(app.recoveryTestControl.workspaces.wait(app.ctx, targetKey))

		// And the recoveries for workspaces 3, 5, 7, and 9 complete successfully without restarting
		for index, key := range oldKeys {
			oldReleases[index]()
			require.NoError(app.recoveryTestControl.workspaces.wait(app.ctx, key))
			require.Equal(1, app.recoveryTestControl.workspaces.startCount(key))
		}
	})

	t.Run("service cancellation stops active workspace attempts", func(t *testing.T) {
		require := require.New(t)
		app, _ := setWorkspaceRecoveryApp(t, 1, 3, 5, 7, 9, 11)
		key := workspaceManagerKey(5)
		rejectedKey := workspaceManagerKey(3)
		app.recoveryTestControl.workspaces.blockNext(key)
		// Occupy the other three slots so workspace 3 is rejected.
		for _, wsid := range []istructs.WSID{7, 9, 11} {
			app.recoveryTestControl.workspaces.blockNext(workspaceManagerKey(wsid))
			status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(wsid, app))
			require.NoError(err)
			require.Equal(http.StatusServiceUnavailable, status)
		}
		status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(5, app))
		require.NoError(err)
		require.Equal(http.StatusServiceUnavailable, status)
		status, err = sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(3, app))
		require.NoError(err)
		require.Equal(http.StatusServiceUnavailable, status)
		require.Equal(0, app.recoveryTestControl.workspaces.startCount(rejectedKey))

		app.cancel()
		<-app.done
		require.ErrorIs(app.recoveryTestControl.workspaces.wait(context.Background(), key), context.Canceled)
		require.Equal(0, app.recoveryTestControl.workspaces.startCount(rejectedKey))
		app.n10nBrokerCleanup()
	})
}

func TestServiceShutdownWaitsForWorkspaceRecovery(t *testing.T) {
	for _, test := range []struct {
		name           string
		resetPartition bool
	}{
		{name: "current partition"},
		{name: "partition detached by reset", resetPartition: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			storage := testutils.NewCheckpointStorageMem()
			var failNextProjector atomic.Bool
			projectorErr := errors.New("reset partition while workspace recovery is running")
			app := setUp(t, func(wsb appdef.IWorkspaceBuilder, cfg *istructsmem.AppConfigType) {
				addTestCUDRecords(wsb)
				addTestCUDCommand(wsb, cfg)
				addStandardTestRoles(wsb)
				projectorName := appdef.NewQName("test", "FailNextProjector")
				cfg.AddSyncProjectors(istructs.Projector{
					Name: projectorName,
					Func: func(istructs.IPLogEvent, istructs.IState, istructs.IIntents) error {
						if failNextProjector.Swap(false) {
							return projectorErr
						}
						return nil
					},
				})
				wsb.AddProjector(projectorName).SetSync(true).Events().Add(
					[]appdef.OperationKind{appdef.OperationKind_Execute}, filter.QNames(istructs.QNameCommandCUD))
			}, withCheckpointStorage(storage), withTestWorkspaces(1, 3))
			stopService := sync.OnceFunc(app.cancel)
			allowWorkerExit := make(chan struct{})
			releaseWorker := sync.OnceFunc(func() { close(allowWorkerExit) })
			t.Cleanup(func() {
				releaseWorker()
				stopService()
				select {
				case <-app.done:
				case <-time.After(5 * time.Second):
					t.Error("command service did not stop after workspace recovery was released")
				}
				app.n10nBrokerCleanup()
			})

			// Recover the shared partition and workspace 3; workspace 1 stays unrecovered.
			sendCUD_3Inserts(t, 3, app)
			key := recoveryWorkspaceKeyForTestApp(app, 1)
			recoveryStarted := make(chan struct{})
			cancellationObserved := make(chan struct{})
			storage.SetBefore(testutils.GetWorkspace, func(op testutils.Operation) error {
				if op.AppID != app.appStructs.ClusterAppID() || op.WSID != key.wsid {
					return nil
				}
				close(recoveryStarted)
				<-app.ctx.Done()
				close(cancellationObserved)
				// Observing cancellation does not mean the worker has finished exiting.
				<-allowWorkerExit
				return app.ctx.Err()
			})
			triggerWorkspaceRecovery(t, app, key.wsid)
			timedReadChan(t, recoveryStarted)

			if test.resetPartition {
				// A real command failure resets the partition while its workspace worker is blocked.
				failNextProjector.Store(true)
				response := sendCUD_3Inserts(t, 3, app, http.StatusInternalServerError)
				require.Equal(t, projectorErr.Error(), response["sys.Error"].(map[string]interface{})["Message"])
				// A subsequent request recovers a replacement partition, detaching the old workspace manager.
				require.NoError(t, triggerAndWaitForRecovery(t, app, 3))
			}

			stopService()
			timedReadChan(t, cancellationObserved)
			// Keep the canceled worker blocked long enough to observe an early service exit.
			select {
			case <-app.done:
				t.Error("command service stopped before its canceled workspace recovery worker exited")
			case <-time.After(100 * time.Millisecond):
			}
			releaseWorker()
			waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			require.ErrorIs(t, app.recoveryTestControl.workspaces.wait(waitCtx, key), context.Canceled)
			timedReadChan(t, app.done)
		})
	}
}

func setWorkspaceRecoveryApp(t *testing.T, lazyWSIDs ...istructs.WSID) (testApp, *testutils.CheckpointStorage) {
	t.Helper()
	require := require.New(t)
	storage := testutils.NewCheckpointStorageMem()
	allWSIDs := append(append([]istructs.WSID(nil), lazyWSIDs...), 99)
	app := setUpRecoveryTestApp(t,
		withCheckpointStorage(storage),
		withTestWorkspaces(allWSIDs...),
	)
	partitionID := coreutils.AppPartitionID(99, testAppPartCount)
	nextPLogOffset := nextPLogOffset(t, app.appStructs, partitionID)
	storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), partitionID,
		checkpoints.PartitionCheckpoint{LastPLogOffset: nextPLogOffset - 1})
	for _, wsid := range lazyWSIDs {
		storage.ForcePutWorkspaceCheckpoint(t, app.appStructs.ClusterAppID(), wsid, checkpoints.WorkspaceCheckpoint{})
	}

	require.NoError(triggerAndWaitForRecovery(t, app, 99))
	triggerWorkspaceRecovery(t, app, 99)
	require.NoError(app.recoveryTestControl.workspaces.wait(app.ctx, workspaceManagerKey(99)))
	return app, storage
}

func setRecoveryAdmissionApp(t *testing.T, wsids ...istructs.WSID) testApp {
	t.Helper()
	require := require.New(t)
	storage := testutils.NewCheckpointStorageMem()
	app := setUpRecoveryTestApp(t,
		withCheckpointStorage(storage),
		withTestAppPartitions(1),
		withTestWorkspaces(wsids...),
	)

	partitionKey := recoveryKeyForTestApp(app, wsids[0])
	status, err := sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(wsids[0], app))
	require.NoError(err)
	require.Equal(http.StatusServiceUnavailable, status)
	require.NoError(app.recoveryTestControl.partitions.wait(app.ctx, partitionKey))

	for _, wsid := range wsids {
		workspaceKey := recoveryWorkspaceKeyForTestApp(app, wsid)
		status, err = sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(wsid, app))
		require.NoError(err)
		require.Equal(http.StatusServiceUnavailable, status)
		require.NoError(app.recoveryTestControl.workspaces.wait(app.ctx, workspaceKey))
		prepareRecoveryAdmissionWLog(t, app, wsid)
		storage.ForcePutWorkspaceCheckpoint(t, app.appStructs.ClusterAppID(), wsid, checkpoints.WorkspaceCheckpoint{
			WLogIDBearingOffset: 10,
		})
	}
	storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), partitionKey.partitionID,
		checkpoints.PartitionCheckpoint{
			LastPLogOffset: nextPLogOffset(t, app.appStructs, partitionKey.partitionID) - 1,
		})

	restartCmdProc(&app)
	status, err = sendRequestForStatus(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(wsids[0], app))
	require.NoError(err)
	require.Equal(http.StatusServiceUnavailable, status)
	require.NoError(app.recoveryTestControl.partitions.wait(app.ctx, partitionKey))
	return app
}

func prepareRecoveryAdmissionWLog(t *testing.T, app testApp, wsid istructs.WSID) {
	t.Helper()
	require := require.New(t)

	for expectedOffset := istructs.Offset(2); expectedOffset <= 5; expectedOffset++ {
		response := sendNoCUDCommand(t, wsid, app)
		require.Equal(int(expectedOffset), int(response["CurrentWLogOffset"].(float64)))
	}
	firstIDBearing := sendCUDWithBody(t, wsid, app,
		`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.TestCDoc"}}]}`)
	require.Equal(6, int(firstIDBearing["CurrentWLogOffset"].(float64)))
	require.Equal(istructs.FirstUserRecordID,
		istructs.RecordID(firstIDBearing["NewIDs"].(map[string]interface{})["1"].(float64)))

	for expectedOffset := istructs.Offset(7); expectedOffset <= 9; expectedOffset++ {
		response := sendNoCUDCommand(t, wsid, app)
		require.Equal(int(expectedOffset), int(response["CurrentWLogOffset"].(float64)))
	}
	lastRecordIDBearing := sendCUDWithBody(t, wsid, app, `{"cuds":[
		{"fields":{"sys.ID":1,"sys.QName":"test.TestCDoc"}},
		{"fields":{"sys.ID":2,"sys.QName":"test.TestCDoc"}},
		{"fields":{"sys.ID":3,"sys.QName":"test.TestCDoc"}},
		{"fields":{"sys.ID":4,"sys.QName":"test.TestCDoc"}}
	]}`)
	require.Equal(10, int(lastRecordIDBearing["CurrentWLogOffset"].(float64)))
	for rawID := 1; rawID <= 4; rawID++ {
		require.Equal(istructs.RecordID(200001+rawID),
			istructs.RecordID(lastRecordIDBearing["NewIDs"].(map[string]interface{})[strconv.Itoa(rawID)].(float64)))
	}

	for expectedOffset := istructs.Offset(11); expectedOffset <= 14; expectedOffset++ {
		response := sendNoCUDCommand(t, wsid, app)
		require.Equal(int(expectedOffset), int(response["CurrentWLogOffset"].(float64)))
		require.Empty(response["NewIDs"])
	}
}

func workspaceManagerKey(wsid istructs.WSID) workspaceKey {
	return workspaceKey{partitionKey: partitionManagerKeyForWSID(wsid), wsid: wsid}
}

func restartAndRecoverWorkspace(t *testing.T, app *testApp) {
	t.Helper()
	restartCmdProc(app)
	app.recoveryTestControl.pLogReads.reset(partitionManagerKeyForWSID(1))
	app.recoveryTestControl.wLogReads.reset(workspaceManagerKey(1))
	require.NoError(t, triggerAndWaitForWorkspaceRecovery(t, *app, 1))
}

func testRecoveryFromInvalidPartitionCheckpoint(t *testing.T, app *testApp, storage *testutils.CheckpointStorage,
	checkpointOffset, expectedNextWLogOffset istructs.Offset, expectedNextRecordID istructs.RecordID) {
	t.Helper()
	logCap := logger.StartCapture(t, logger.LogLevelVerbose)
	key := partitionManagerKeyForWSID(1)
	storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), key.partitionID,
		checkpoints.PartitionCheckpoint{LastPLogOffset: checkpointOffset})
	restartAndRecoverWorkspace(t, app)
	requireCheckpointFallbackReads(t, checkpointOffset, app.recoveryTestControl.pLogReads.values(key))
	requireLogError(logCap, "cp.partition_recovery.checkpoint.invalid",
		fmt.Sprintf("stored PLog offset %d does not identify an existing event; retrying from FirstOffset", checkpointOffset))
	requireNextCommandNumbers(t, *app, expectedNextWLogOffset, expectedNextRecordID)
}

func testRecoveryFromInvalidWorkspaceCheckpoint(t *testing.T, app *testApp, storage *testutils.CheckpointStorage,
	checkpointPLogOffset, checkpointWLogOffset, expectedNextWLogOffset istructs.Offset, expectedNextRecordID istructs.RecordID) {
	t.Helper()
	logCap := logger.StartCapture(t, logger.LogLevelVerbose)
	key := partitionManagerKeyForWSID(1)
	workspaceKey := workspaceManagerKey(1)
	storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), key.partitionID, checkpoints.PartitionCheckpoint{LastPLogOffset: checkpointPLogOffset})
	storage.ForcePutWorkspaceCheckpoint(t, app.appStructs.ClusterAppID(), 1, checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: checkpointWLogOffset})
	restartAndRecoverWorkspace(t, app)
	requireCheckpointFallbackReads(t, checkpointWLogOffset, app.recoveryTestControl.wLogReads.values(workspaceKey))
	requireLogError(logCap, "cp.workspace_recovery.checkpoint.invalid",
		fmt.Sprintf("stored WLog offset %d does not identify an ID-bearing event; retrying from FirstOffset", checkpointWLogOffset))
	requireNextCommandNumbers(t, *app, expectedNextWLogOffset, expectedNextRecordID)
}

func testRecoveryFromCheckpoints(t *testing.T, app *testApp, storage *testutils.CheckpointStorage,
	expetedPLogOffset_From, expectedWLogOffset_From, expectedNextWLogOffset istructs.Offset, expectedNextRecordID istructs.RecordID) {
	t.Helper()
	logCap := logger.StartCapture(t, logger.LogLevelVerbose)
	key := partitionManagerKeyForWSID(1)
	workspaceKey := workspaceManagerKey(1)
	storage.ForcePutWorkspaceCheckpoint(t, app.appStructs.ClusterAppID(), 1,
		checkpoints.WorkspaceCheckpoint{WLogIDBearingOffset: expectedWLogOffset_From})
	storage.ForcePutPartitionCheckpoint(t, app.appStructs.ClusterAppID(), key.partitionID,
		checkpoints.PartitionCheckpoint{LastPLogOffset: expetedPLogOffset_From})

	restartAndRecoverWorkspace(t, app)

	// A zero checkpoint starts at FirstOffset; nonzero checkpoints are inclusive.
	if expetedPLogOffset_From == 0 {
		expetedPLogOffset_From = istructs.FirstOffset
	}
	if expectedWLogOffset_From == 0 {
		expectedWLogOffset_From = istructs.FirstOffset
	}
	require.Equal(t, []logReadFromTo{{offset: expetedPLogOffset_From, count: istructs.ReadToTheEnd}},
		app.recoveryTestControl.pLogReads.values(key))
	require.Equal(t, []logReadFromTo{{offset: expectedWLogOffset_From, count: istructs.ReadToTheEnd}},
		app.recoveryTestControl.wLogReads.values(workspaceKey))
	logCap.NotContains("cp.partition_recovery.checkpoint.invalid",
		"cp.workspace_recovery.checkpoint.invalid")
	requireNextCommandNumbers(t, *app, expectedNextWLogOffset, expectedNextRecordID)
}

func requireCommandError(t *testing.T, meta bus.ResponseMeta, err error, expectedStatus int, expectedMessage string) {
	t.Helper()
	require.Equal(t, expectedStatus, meta.StatusCode)
	require.Equal(t, httpu.ContentType_ApplicationJSON, meta.ContentType)
	if err == nil {
		require.Empty(t, expectedMessage)
		return
	}
	var sysError coreutils.SysError
	require.ErrorAs(t, err, &sysError)
	require.Equal(t, expectedStatus, sysError.HTTPStatus)
	require.Contains(t, sysError.Message, expectedMessage)
}

func requireIssuedNumbers(t *testing.T, response map[string]interface{}, expectedWLogOffset istructs.Offset, expectedRecordID istructs.RecordID) {
	t.Helper()
	require.Equal(t, expectedWLogOffset, istructs.Offset(response["CurrentWLogOffset"].(float64)))
	require.Equal(t, expectedRecordID,
		istructs.RecordID(response["NewIDs"].(map[string]interface{})["1"].(float64)))
}

// requireNextCommandNumbers verifies that a new insert continues the recovered offsets and record IDs.
func requireNextCommandNumbers(t *testing.T, app testApp, expectedWLogOffset istructs.Offset, expectedRecordID istructs.RecordID) {
	t.Helper()
	response := sendCUD_3Inserts(t, 1, app)
	requireIssuedNumbers(t, response, expectedWLogOffset, expectedRecordID)
}

func nextPLogOffset(t *testing.T, appStructs istructs.IAppStructs, partitionID istructs.PartitionID) istructs.Offset {
	t.Helper()
	next := istructs.FirstOffset
	err := appStructs.Events().ReadPLog(t.Context(), partitionID, istructs.FirstOffset, istructs.ReadToTheEnd,
		func(offset istructs.Offset, event istructs.IPLogEvent) error {
			event.Release()
			next = offset + 1
			return nil
		})
	require.NoError(t, err)
	return next
}

func requireCheckpointFallbackReads(t *testing.T, checkpointOffset istructs.Offset, actual []logReadFromTo) {
	t.Helper()
	require.Equal(t, []logReadFromTo{
		{offset: checkpointOffset, count: istructs.ReadToTheEnd},
		{offset: istructs.FirstOffset, count: istructs.ReadToTheEnd},
	}, actual)
}

func requireLogError(logCap logger.ILogCaptor, stage, message string) {
	logCap.HasLine("level=ERROR", "stage="+stage, fmt.Sprintf("msg=%q", message))
}

func countCapturedLogLines(logCap logger.ILogCaptor, substrings ...string) int {
	count := 0
	for line := range strings.SplitSeq(logCap.String(), "\n") {
		matches := true
		for _, substring := range substrings {
			if !strings.Contains(line, substring) {
				matches = false
				break
			}
		}
		if matches {
			count++
		}
	}
	return count
}

type appStructsWithTestEvents struct {
	istructs.IAppStructs
	events istructs.IEvents
}

func (a *appStructsWithTestEvents) Events() istructs.IEvents {
	return a.events
}

// pLogCheckpointSkippingEvents simulates a deleted checkpoint event while
// retaining readable events after it. The fallback read from FirstOffset is
// delegated unchanged so recovery can reconstruct the actual log end.
type pLogCheckpointSkippingEvents struct {
	istructs.IEvents
	partitionID      istructs.PartitionID
	checkpointOffset istructs.Offset
}

func (e *pLogCheckpointSkippingEvents) ReadPLog(ctx context.Context, partitionID istructs.PartitionID,
	offset istructs.Offset, toReadCount int, cb istructs.PLogEventsReaderCallback) error {
	if partitionID == e.partitionID && offset == e.checkpointOffset {
		offset++
	}
	return e.IEvents.ReadPLog(ctx, partitionID, offset, toReadCount, cb)
}

type wLogCheckpointSkippingEvents struct {
	istructs.IEvents
	wsid             istructs.WSID
	checkpointOffset istructs.Offset
}

func (e *wLogCheckpointSkippingEvents) ReadWLog(ctx context.Context, wsid istructs.WSID,
	offset istructs.Offset, toReadCount int, cb istructs.WLogEventsReaderCallback) error {
	if wsid == e.wsid && offset == e.checkpointOffset {
		offset++
	}
	return e.IEvents.ReadWLog(ctx, wsid, offset, toReadCount, cb)
}

func newDirectRecoveryCmdProc(storage checkpoints.IRecoveryCheckpointStorage, control *recoveryTestControl) *cmdProc {
	partitionHooks, workspaceHooks := control.testHooks()
	return &cmdProc{
		checkpointStorage:      storage,
		partitionRecoveryHooks: hooksOrNOP(partitionHooks),
		workspaceRecoveryHooks: hooksOrNOP(workspaceHooks),
		storeOp:                &pipeline.NOOP{},
	}
}

func TestCUDUpdate(t *testing.T) {
	require := require.New(t)

	testQName := appdef.NewQName("test", "test")

	app := setUp(t, func(wsb appdef.IWorkspaceBuilder, cfg *istructsmem.AppConfigType) {
		wsb.AddCDoc(testQName).AddField("IntFld", appdef.DataKind_int32, false)
		addTestCUDCommand(wsb, cfg)
		addStandardTestRoles(wsb)
	})
	defer tearDown(app)

	// insert
	req := bus.Request{
		WSID:     1,
		AppQName: istructs.AppQName_untill_airs_bp,
		Resource: "c.sys.CUD",
		Body:     []byte(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.test"}}]}`),
		Header:   app.sysAuthHeader,
	}
	cmdRespMeta, cmdResp, sysErr := bus.GetCommandResponse(app.ctx, app.requestSender, req)
	require.NoError(sysErr)
	require.Equal(http.StatusOK, cmdRespMeta.StatusCode)
	require.Equal(httpu.ContentType_ApplicationJSON, cmdRespMeta.ContentType)
	require.Empty(cmdResp.CmdResult)
	newID := cmdResp.NewIDs["1"]
	require.NotZero(newID)

	t.Run("update", func(t *testing.T) {
		req.Body = fmt.Appendf(nil, `{"cuds":[{"sys.ID":%d,"fields":{"sys.QName":"test.test", "IntFld": 42}}]}`, newID)
		cmdRespMeta, _, err := bus.GetCommandResponse(app.ctx, app.requestSender, req)
		require.NoError(err)
		require.Equal(http.StatusOK, cmdRespMeta.StatusCode)
		require.Equal(httpu.ContentType_ApplicationJSON, cmdRespMeta.ContentType)
	})

	t.Run("404 not found on update not existing", func(t *testing.T) {
		req.Body = fmt.Appendf(nil, `{"cuds":[{"sys.ID":%d,"fields":{"sys.QName":"test.test", "IntFld": 42}}]}`, istructs.NonExistingRecordID)
		cmdRespMeta, _, err := bus.GetCommandResponse(app.ctx, app.requestSender, req)
		require.Error(err)
		require.Equal(http.StatusNotFound, cmdRespMeta.StatusCode)
		require.Equal(httpu.ContentType_ApplicationJSON, cmdRespMeta.ContentType)
	})
}

func Test400BadRequestOnCUDErrors(t *testing.T) {
	testQName := appdef.NewQName("test", "test")

	app := setUp(t, func(wsb appdef.IWorkspaceBuilder, cfg *istructsmem.AppConfigType) {
		wsb.AddCDoc(testQName)
		addTestCUDCommand(wsb, cfg)
		addStandardTestRoles(wsb)
	})
	defer tearDown(app)

	cases := []struct {
		desc                string
		bodyAdd             string
		expectedMessageLike string
	}{
		{"not an object", `"cuds":42`, `field "cuds" must be an array of objects`},
		{`element is not an object`, `"cuds":[42]`, `cuds[0]: not an object`},
		{`missing fields`, `"cuds":[{}]`, `cuds[0]: "fields" missing`},
		{`fields is not an object`, `"cuds":[{"fields":42}]`, `cuds[0]: field "fields" must be an object`},
		{`fields: sys.ID missing`, `"cuds":[{"fields":{"sys.QName":"test.Test"}}]`, `cuds[0]: "sys.ID" field missing`},
		{`fields: sys.ID is not a number (create)`, `"cuds":[{"sys.ID":"wrong","fields":{"sys.QName":"test.Test"}}]`, `cuds[0]: field "sys.ID" must be json.Number`},
		{`fields: sys.ID is not a number (update)`, `"cuds":[{"fields":{"sys.ID":"wrong","sys.QName":"test.Test"}}]`, `cuds[0]: field "sys.ID" must be json.Number`},
		{`fields: wrong qName`, `"cuds":[{"fields":{"sys.ID":1,"sys.QName":"wrong"}},{"fields":{"sys.ID":1,"sys.QName":"test.Test"}}]`, `convert error: string «wrong»`},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			req := bus.Request{
				WSID:     1,
				AppQName: istructs.AppQName_untill_airs_bp,
				Resource: "c.sys.CUD",
				Body:     []byte("{" + c.bodyAdd + "}"),
				Header:   app.sysAuthHeader,
			}
			cmdRespMeta, _, sysErr := bus.GetCommandResponse(app.ctx, app.requestSender, req)
			requireCommandError(t, cmdRespMeta, sysErr, http.StatusBadRequest, c.expectedMessageLike)
		})
	}
}

func TestErrors(t *testing.T) {
	testCmdQNameParams := appdef.NewQName(appdef.SysPackage, "TestParams")
	testCmdQNameParamsUnlogged := appdef.NewQName(appdef.SysPackage, "TestParamsUnlogged")

	testCmdQName := appdef.NewQName(appdef.SysPackage, "Test")
	app := setUp(t, func(wsb appdef.IWorkspaceBuilder, cfg *istructsmem.AppConfigType) {
		wsb.AddObject(testCmdQNameParams).
			AddField("Text", appdef.DataKind_string, true)

		wsb.AddObject(testCmdQNameParamsUnlogged).
			AddField("Password", appdef.DataKind_string, true)

		wsb.AddCommand(testCmdQName).SetUnloggedParam(testCmdQNameParamsUnlogged).SetParam(testCmdQNameParams)

		addStandardTestRoles(wsb)

		cfg.Resources.Add(istructsmem.NewCommandFunction(testCmdQName, istructsmem.NullCommandExec))
	})
	defer tearDown(app)

	baseReq := bus.Request{
		WSID:     1,
		AppQName: istructs.AppQName_untill_airs_bp,
		Resource: "c.sys.Test",
		Body:     []byte(`{"args":{"Text":"hello"},"unloggedArgs":{"Password":"123"}}`),
		Header:   app.sysAuthHeader,
	}

	cases := []struct {
		desc string
		bus.Request
		expectedMessageLike string
		expectedStatusCode  int
	}{
		{"unknown app", bus.Request{AppQName: appdef.NewAppQName("untill", "unknown")}, "application not found", http.StatusBadRequest},
		{"bad request body", bus.Request{Body: []byte("{wrong")}, "failed to unmarshal request body: invalid character 'w' looking for beginning of object key string", http.StatusBadRequest},
		{"unknown func", bus.Request{Resource: "c.sys.Unknown"}, "unknown function", http.StatusBadRequest},
		{"args: field of wrong type provided", bus.Request{Body: []byte(`{"args":{"Text":42}}`)}, "wrong field type", http.StatusBadRequest},
		{"args: not an object", bus.Request{Body: []byte(`{"args":42}`)}, `field "args" must be an object`, http.StatusBadRequest},
		{"args: missing at all with a required field", bus.Request{Body: []byte(`{}`)}, "", http.StatusBadRequest},
		{"unloggedArgs: not an object", bus.Request{Body: []byte(`{"unloggedArgs":42,"args":{"Text":"txt"}}`)}, `field "unloggedArgs" must be an object`, http.StatusBadRequest},
		{"unloggedArgs: field of wrong type provided", bus.Request{Body: []byte(`{"unloggedArgs":{"Password":42},"args":{"Text":"txt"}}`)}, "wrong field type", http.StatusBadRequest},
		{"unloggedArgs: missing required field of unlogged args, no unlogged args at all", bus.Request{Body: []byte(`{"args":{"Text":"txt"}}`)}, "", http.StatusBadRequest},
		{"cuds: not an object", bus.Request{Body: []byte(`{"args":{"Text":"hello"},"unloggedArgs":{"Password":"123"},"cuds":42}`)}, `field "cuds" must be an array of objects`, http.StatusBadRequest},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			req := baseReq
			req.Body = make([]byte, len(baseReq.Body))
			copy(req.Body, baseReq.Body)
			if c.AppQName != appdef.NullAppQName {
				req.AppQName = c.AppQName
			}
			if len(c.Body) > 0 {
				req.Body = make([]byte, len(c.Body))
				copy(req.Body, c.Body)
			}
			if len(c.Resource) > 0 {
				req.Resource = c.Resource
			}
			cmdRespMeta, _, sysErr := bus.GetCommandResponse(app.ctx, app.requestSender, req)
			requireCommandError(t, cmdRespMeta, sysErr, c.expectedStatusCode, c.expectedMessageLike)
		})
	}
}

// Only Borrow and the registration lookup are available: any attempt to access
// runtime partition state through this double fails the test immediately.
type failingBorrowAppParts struct {
	appparts.IAppPartitions
	borrowErr         error
	registrationErr   error
	registrationCalls int
}

func (p *failingBorrowAppParts) Borrow(appdef.AppQName, istructs.PartitionID, appparts.ProcessorKind) (appparts.IAppPartition, error) {
	return nil, p.borrowErr
}

func (p *failingBorrowAppParts) AppPartsCount(appdef.AppQName) (istructs.NumAppPartitions, error) {
	p.registrationCalls++
	if p.registrationErr != nil {
		return 0, p.registrationErr
	}
	return 2, nil
}

func TestBorrowErrorResponseAndLogging(t *testing.T) {
	for _, test := range []struct {
		name            string
		borrowErr       error
		registrationErr error
		status          int
		message         string
	}{
		{"missing application", fmt.Errorf("opaque lookup: %w", appparts.ErrNotFound), appparts.ErrNotFound,
			http.StatusBadRequest, "application not found"},
		{"missing partition", fmt.Errorf("opaque lookup: %w", appparts.ErrNotFound), nil,
			http.StatusInternalServerError, "partition not found"},
		{"engine exhaustion", appparts.ErrNotAvailableEngines, nil,
			http.StatusServiceUnavailable, "no available engines"},
	} {
		t.Run(test.name, func(t *testing.T) {
			parts := &failingBorrowAppParts{borrowErr: test.borrowErr, registrationErr: test.registrationErr}
			ctx := logger.WithContextAttrs(t.Context(), map[string]any{
				logger.LogAttr_ReqID:     "borrow-test",
				logger.LogAttr_WSID:      istructs.WSID(42),
				logger.LogAttr_Extension: "c.sys.CUD",
			})
			cmd := &cmdWorkpiece{
				appParts: parts,
				cmdMes: NewCommandMessage(ctx, []byte("{}"), testAppName, 42, nil, 1,
					istructs.QNameCommandCUD, "", "", 0, 0, "", ""),
			}
			logCap := logger.StartCapture(t, logger.LogLevelVerbose)
			for range 2 {
				err := cmd.borrow()
				var responseError coreutils.SysError
				require.ErrorAs(t, err, &responseError)
				require.Equal(t, test.status, responseError.HTTPStatus)
				require.Equal(t, test.message, responseError.Message)
				require.NotContains(t, responseError.Message, testAppName.String())
				require.NotContains(t, responseError.Message, "opaque lookup")
				require.Nil(t, cmd.appPart)
				logHandlingError(cmd, err)
			}
			if errors.Is(test.borrowErr, appparts.ErrNotFound) {
				require.Equal(t, 2, parts.registrationCalls, "classify by registration, not error text")
			} else {
				require.Zero(t, parts.registrationCalls, "engine exhaustion is not a registration failure")
			}
			logCap.HasLine("stage=cp.error", test.message, "vapp="+testAppName.String(), "partid=1",
				"reqid=borrow-test", "wsid=42", "extension=c.sys.CUD")
			logCap.NotContains("cp.partition_recovery", "cp.workspace_recovery", "sys._Recovery")
		})
	}
}

func TestRejectedCommandsDoNotRecoverOrWrite(t *testing.T) {
	parts := &failingBorrowAppParts{borrowErr: appparts.ErrNotFound}
	app := setUpRecoveryTestApp(t, func(options *testAppOptions) {
		options.appPartsForProcessor = func(appparts.IAppPartitions) appparts.IAppPartitions { return parts }
	})
	defer tearDown(app)
	partitionID := coreutils.AppPartitionID(1, app.numAppPartitions)
	readOffsets := func() (plog, wlog []istructs.Offset) {
		require.NoError(t, app.appStructs.Events().ReadPLog(app.ctx, partitionID, istructs.FirstOffset,
			istructs.ReadToTheEnd, func(offset istructs.Offset, _ istructs.IPLogEvent) error {
				plog = append(plog, offset)
				return nil
			}))
		require.NoError(t, app.appStructs.Events().ReadWLog(app.ctx, 1, istructs.FirstOffset,
			istructs.ReadToTheEnd, func(offset istructs.Offset, _ istructs.IWLogEvent) error {
				wlog = append(wlog, offset)
				return nil
			}))
		return
	}
	beforePLog, beforeWLog := readOffsets()
	logCap := logger.StartCapture(t, logger.LogLevelVerbose)
	for range 3 {
		meta, _, err := bus.GetCommandResponse(app.ctx, app.rawRequestSender, newCUD_3Inserts_Request(1, app))
		require.Equal(t, http.StatusInternalServerError, meta.StatusCode)
		var responseError coreutils.SysError
		require.ErrorAs(t, err, &responseError)
		require.Equal(t, "partition not found", responseError.Message)
	}
	afterPLog, afterWLog := readOffsets()
	require.Equal(t, beforePLog, afterPLog)
	require.Equal(t, beforeWLog, afterWLog)
	require.Zero(t, app.recoveryTestControl.partitions.startCount(recoveryKeyForTestApp(app, 1)))
	require.Zero(t, app.recoveryTestControl.workspaces.startCount(recoveryWorkspaceKeyForTestApp(app, 1)))
	logCap.NotContains("cp.partition_recovery", "cp.workspace_recovery", "cp.plog_saved")
}

func TestAuthnz(t *testing.T) {
	require := require.New(t)

	qNameTestDeniedCDoc := appdef.NewQName("app1pkg", "TestDeniedCDoc") // the same in core/iauthnzimpl

	qNameAllowedCmd := appdef.NewQName(appdef.SysPackage, "TestAllowedCmd")
	qNameDeniedCmd := appdef.NewQName(appdef.SysPackage, "TestDeniedCmd") // the same in core/iauthnzimpl
	app := setUp(t, func(wsb appdef.IWorkspaceBuilder, cfg *istructsmem.AppConfigType) {
		wsb.AddCDoc(qNameTestDeniedCDoc)
		wsb.AddCommand(qNameAllowedCmd)
		wsb.AddCommand(qNameDeniedCmd)
		addTestCUDCommand(wsb, cfg)
		addStandardTestRoles(wsb)
		wsb.AddRole(iauthnz.QNameRoleProfileOwner)
		wsb.AddRole(iauthnz.QNameRoleAnonymous)
		wsb.AddRole(iauthnz.QNameRoleWorkspaceOwner)
		cfg.Resources.Add(istructsmem.NewCommandFunction(qNameAllowedCmd, istructsmem.NullCommandExec))
		cfg.Resources.Add(istructsmem.NewCommandFunction(qNameDeniedCmd, istructsmem.NullCommandExec))

		wsb.Revoke([]appdef.OperationKind{appdef.OperationKind_Execute}, filter.QNames(qNameDeniedCmd), nil, iauthnz.QNameRoleWorkspaceOwner)
	})
	defer tearDown(app)

	pp := payloads.PrincipalPayload{
		Login:       "testlogin",
		SubjectKind: istructs.SubjectKind_User,
		ProfileWSID: 1,
	}
	token, err := app.appTokens.IssueToken(10*time.Second, &pp)
	require.NoError(err)

	type testCase struct {
		desc               string
		req                bus.Request
		expectedStatusCode int
	}
	cases := []testCase{
		{
			desc: "403 on cmd EXECUTE forbidden", req: bus.Request{
				Body:     []byte(`{}`),
				AppQName: istructs.AppQName_untill_airs_bp,
				WSID:     1,
				Resource: "c.sys.TestDeniedCmd",
				Header:   getAuthHeader(token),
			},
			expectedStatusCode: http.StatusForbidden,
		},
		{
			desc: "403 on INSERT CUD forbidden", req: bus.Request{
				Body:     []byte(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.TestDeniedCDoc"}}]}`),
				AppQName: istructs.AppQName_untill_airs_bp,
				WSID:     1,
				Resource: "c.sys.CUD",
				Header:   getAuthHeader(token),
			},
			expectedStatusCode: http.StatusForbidden,
		},
		{
			desc: "403 if no token for a func that requires authentication", req: bus.Request{
				Body:     []byte(`{}`),
				AppQName: istructs.AppQName_untill_airs_bp,
				WSID:     1,
				Resource: "c.sys.TestAllowedCmd",
			},
			expectedStatusCode: http.StatusForbidden,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			cmdRespMeta, _, err := bus.GetCommandResponse(app.ctx, app.requestSender, c.req)
			require.Error(err)
			require.Equal(c.expectedStatusCode, cmdRespMeta.StatusCode, c.desc)
		})
	}
}

func getAuthHeader(token string) map[string]string {
	return map[string]string{
		httpu.Authorization: "Bearer " + token,
	}
}

func TestBasicUsage_FuncWithRawArg(t *testing.T) {
	require := require.New(t)
	testCmdQName := appdef.NewQName(appdef.SysPackage, "Test")
	ch := make(chan interface{})
	app := setUp(t, func(wsb appdef.IWorkspaceBuilder, cfg *istructsmem.AppConfigType) {
		wsb.AddCommand(testCmdQName).SetParam(istructs.QNameRaw)
		addStandardTestRoles(wsb)
		cfg.Resources.Add(istructsmem.NewCommandFunction(testCmdQName, func(args istructs.ExecCommandArgs) (err error) {
			require.Equal("custom content", args.ArgumentObject.AsString(processors.Field_RawObject_Body))
			close(ch)
			return
		}))
	})
	defer tearDown(app)

	request := bus.Request{
		Body:     []byte(`custom content`),
		AppQName: istructs.AppQName_untill_airs_bp,
		WSID:     1,
		Resource: "c.sys.Test",
		Header:   app.sysAuthHeader,
	}
	cmdRespMeta, _, err := bus.GetCommandResponse(app.ctx, app.requestSender, request)
	require.NoError(err)
	require.Equal(http.StatusOK, cmdRespMeta.StatusCode)
	require.Equal(httpu.ContentType_ApplicationJSON, cmdRespMeta.ContentType)
	<-ch
}

func TestRateLimit(t *testing.T) {
	require := require.New(t)

	qName := appdef.NewQName(appdef.SysPackage, "MyCmd")
	parsQName := appdef.NewQName(appdef.SysPackage, "Params")

	app := setUp(t,
		func(wsb appdef.IWorkspaceBuilder, cfg *istructsmem.AppConfigType) {
			addTestCUDRecords(wsb)
			addTestCUDCommand(wsb, cfg)
			wsb.AddObject(parsQName)
			wsb.AddCommand(qName).SetParam(parsQName)
			addStandardTestRoles(wsb)
			cfg.Resources.Add(istructsmem.NewCommandFunction(qName, istructsmem.NullCommandExec))

			rateName := appdef.NewQName(appdef.SysPackage, "MyCmdRate")
			wsb.AddRate(rateName, 2, time.Minute, []appdef.RateScope{appdef.RateScope_Workspace})
			wsb.AddLimit(appdef.NewQName(appdef.SysPackage, "MyCmdLimit"),
				[]appdef.OperationKind{appdef.OperationKind_Execute}, appdef.LimitFilterOption_EACH,
				filter.QNames(qName), rateName)
		})
	defer tearDown(app)
	require.NoError(triggerAndWaitForWorkspaceRecovery(t, app, 1))

	request := bus.Request{
		Body:     []byte(`{"args":{}}`),
		AppQName: istructs.AppQName_untill_airs_bp,
		WSID:     1,
		Resource: "c.sys.MyCmd",
		Header:   app.sysAuthHeader,
	}

	// first 2 calls are ok
	for range 2 {
		cmdRespMeta, _, err := bus.GetCommandResponse(app.ctx, app.requestSender, request)
		require.NoError(err)
		require.Equal(http.StatusOK, cmdRespMeta.StatusCode)
	}

	// 3rd exceeds rate limits
	cmdRespMeta, _, err := bus.GetCommandResponse(app.ctx, app.requestSender, request)
	require.Error(err)
	require.Equal(http.StatusTooManyRequests, cmdRespMeta.StatusCode)

	// 4th call: inspect the SysError directly to assert Retry-After header: period/count = 60s/2 = 30s
	respCh, _, _, err := app.requestSender.SendRequest(app.ctx, request)
	require.NoError(err)
	var sysErr coreutils.SysError
	for elem := range respCh {
		if se, ok := elem.(coreutils.SysError); ok {
			sysErr = se
		}
	}
	require.Equal(http.StatusTooManyRequests, sysErr.HTTPStatus)
	require.Equal("30", sysErr.Headers()[httpu.RetryAfter])
}

type testApp struct {
	ctx                 context.Context
	cfg                 *istructsmem.AppConfigType
	cancel              context.CancelFunc
	done                chan struct{}
	cmdProcService      pipeline.IService
	serviceChannel      CommandChannel
	n10nBroker          in10n.IN10nBroker
	n10nBrokerCleanup   func()
	requestSender       bus.IRequestSender
	rawRequestSender    bus.IRequestSender
	recoveryTestControl *recoveryTestControl
	appStructs          istructs.IAppStructs
	numAppPartitions    istructs.NumAppPartitions

	appTokens     istructs.IAppTokens
	sysAuthHeader map[string]string
}

type testAppOptions struct {
	checkpointStorage    checkpoints.IRecoveryCheckpointStorage
	workspaceIDs         []istructs.WSID
	numAppPartitions     istructs.NumAppPartitions
	appPartsForProcessor func(appparts.IAppPartitions) appparts.IAppPartitions
}

type testAppOption func(*testAppOptions)

func withCheckpointStorage(storage checkpoints.IRecoveryCheckpointStorage) testAppOption {
	return func(options *testAppOptions) { options.checkpointStorage = storage }
}

func withTestWorkspaces(wsids ...istructs.WSID) testAppOption {
	return func(options *testAppOptions) { options.workspaceIDs = wsids }
}

func withTestAppPartitions(num istructs.NumAppPartitions) testAppOption {
	return func(options *testAppOptions) { options.numAppPartitions = num }
}

func tearDown(app testApp) {
	// finish the command processors IService
	app.n10nBrokerCleanup()
	app.cancel()
	<-app.done
}

// test app deployment constants
var (
	testAppName                                = istructs.AppQName_untill_airs_bp
	testAppEngines                             = [appparts.ProcessorKind_Count]uint{10, 10, 10, 0}
	testAppPartCount istructs.NumAppPartitions = 2
)

func setUp(t *testing.T, prepare func(wsb appdef.IWorkspaceBuilder, cfg *istructsmem.AppConfigType), optionFuncs ...testAppOption) testApp {
	require := require.New(t)
	options := testAppOptions{
		checkpointStorage: testutils.NewCheckpointStorageMem(),
		workspaceIDs:      []istructs.WSID{1, 2},
		numAppPartitions:  testAppPartCount,
	}
	for _, optionFunc := range optionFuncs {
		optionFunc(&options)
	}
	// command processor is a IService working through CommandChannel(iprocbus.ServiceChannel). Let's prepare that channel
	serviceChannel := make(CommandChannel)
	done := make(chan struct{})

	vvmCtx, cancel := context.WithCancel(context.Background())
	tm := timeu.NewITime()

	cfgs := istructsmem.AppConfigsType{}
	asf := mem.Provide(testingu.MockTime)
	appStorageProvider := istorageimpl.Provide(asf)

	// build application
	adb := builder.New()
	adb.AddPackage("test", "test.com/test")

	qNameTestWS, qNameTestWSKind := appdef.NewQName(appdef.SysPackage, "TestWS"), appdef.NewQName(appdef.SysPackage, "TestWSKind")
	wsb := adb.AddWorkspace(qNameTestWS)
	wsb.AddCDoc(qNameTestWSKind).SetSingleton()
	wsb.SetDescriptor(qNameTestWSKind)

	wsdescutil.AddWorkspaceDescriptorStubDef(wsb)

	wsb.AddObject(istructs.QNameRaw).AddField(processors.Field_RawObject_Body, appdef.DataKind_string, true, constraints.MaxLen(appdef.MaxFieldLength))

	statelessResources := istructsmem.NewStatelessResources()
	cfg := cfgs.AddBuiltInAppConfig(istructs.AppQName_untill_airs_bp, adb)
	cfg.SetNumAppWorkspaces(istructs.DefaultNumAppWorkspaces)
	if prepare != nil {
		prepare(wsb, cfg)
	}

	appDef, err := adb.Build()
	require.NoError(err)

	appStructsProvider := istructsmem.Provide(cfgs,
		payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT()), appStorageProvider, istructs.SequencesTrustLevel_0, nil)

	secretReader := isecretsimpl.ProvideSecretReader()
	n10nBroker, n10nBrokerCleanup := in10nmem.NewN10nBroker(in10n.Quotas{
		Channels:                1000,
		ChannelsPerSubject:      10,
		Subscriptions:           1000,
		SubscriptionsPerSubject: 10,
	}, tm)

	// prepare the AppParts to borrow AppStructs
	appParts, appPartsClean, err := appparts.New2(vvmCtx, appStructsProvider,
		actualizers.NewSyncActualizerFactoryFactory(actualizers.ProvideSyncActualizerFactory(), secretReader, n10nBroker, statelessResources),
		appparts.NullActualizerRunner,
		appparts.NullSchedulerRunner,
		engines.ProvideExtEngineFactories(
			engines.ExtEngineFactoriesConfig{
				AppConfigs:         cfgs,
				StatelessResources: statelessResources,
				WASMConfig:         iextengine.WASMFactoryConfig{Compile: false},
			}, "", imetrics.Provide()),
		iratesce.TestBucketsFactory)
	require.NoError(err)

	appParts.DeployApp(testAppName, nil, appDef, options.numAppPartitions, testAppEngines, cfg.NumAppWorkspaces())
	partitionIDs := make([]istructs.PartitionID, options.numAppPartitions)
	for partitionID := range options.numAppPartitions {
		partitionIDs[partitionID] = istructs.PartitionID(partitionID) // nolint:gosec
	}
	appParts.DeployAppPartitions(testAppName, partitionIDs)

	// command processor works through ibus.SendResponse -> we need ibus implementation

	rawRequestSender := bus.NewIRequestSender(testingu.MockTime, func(requestCtx context.Context, request bus.Request, responder bus.IResponder) {
		// simulate handling the command request be a real application
		cmdQName, err := appdef.ParseQName(request.Resource[2:])
		require.NoError(err)
		tp := appDef.Type(cmdQName)
		if tp.Kind() == appdef.TypeKind_null {
			bus.ReplyBadRequest(responder, "unknown function")
			return
		}
		token := ""
		if authHeader, ok := request.Header[httpu.Authorization]; ok {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
		partitionID := coreutils.AppPartitionID(request.WSID, options.numAppPartitions)
		icm := NewCommandMessage(requestCtx, request.Body, request.AppQName, request.WSID, responder, partitionID, cmdQName, token, "", 0, 0, "", "")
		serviceChannel <- icm
	})

	tokens := itokensjwt.TestTokensJWT()
	appTokens := payloads.ProvideIAppTokensFactory(tokens).New(testAppName)
	systemToken, err := payloads.GetSystemPrincipalTokenApp(appTokens)
	require.NoError(err)
	recoveryControl := newRecoveryTestControl()
	partitionRecoveryHooks, workspaceRecoveryHooks := recoveryControl.testHooks()
	processorParts := appParts
	if options.appPartsForProcessor != nil {
		processorParts = options.appPartsForProcessor(appParts)
	}
	cmdProcessorFactory := provideServiceFactory(processorParts, tm, n10nBroker, imetrics.Provide(), "vvm",
		iauthnzimpl.NewDefaultAuthenticator(iauthnzimpl.TestSubjectRolesGetter, iauthnzimpl.TestIsDeviceAllowedFuncs), secretReader,
		options.checkpointStorage, partitionRecoveryHooks, workspaceRecoveryHooks)
	cmdProcService := cmdProcessorFactory(serviceChannel)
	requestSender := &recoveryRetrySender{
		raw:     rawRequestSender,
		control: recoveryControl,
		keyForRequest: func(req bus.Request) (partitionKey, bool) {
			return partitionKey{
				appQName:    testAppName,
				partitionID: coreutils.AppPartitionID(req.WSID, options.numAppPartitions),
			}, req.AppQName == testAppName
		},
	}

	go func() {
		cmdProcService.Run(vvmCtx)
		close(done)
	}()

	as, err := appStructsProvider.BuiltIn(istructs.AppQName_untill_airs_bp)
	require.NoError(err)
	for _, wsid := range options.workspaceIDs {
		partitionID := coreutils.AppPartitionID(wsid, options.numAppPartitions)
		err = wsdescutil.CreateCDocWorkspaceDescriptorStub(as, partitionID, wsid, qNameTestWSKind, istructs.Offset(wsid), 1)
		require.NoError(err)
	}

	return testApp{
		cfg:                 cfg,
		requestSender:       requestSender,
		rawRequestSender:    rawRequestSender,
		recoveryTestControl: recoveryControl,
		appStructs:          as,
		numAppPartitions:    options.numAppPartitions,
		cancel:              func() { cancel(); appPartsClean() },
		ctx:                 vvmCtx,
		done:                done,
		cmdProcService:      cmdProcService,
		serviceChannel:      serviceChannel,
		n10nBroker:          n10nBroker,
		n10nBrokerCleanup:   n10nBrokerCleanup,
		appTokens:           appTokens,
		sysAuthHeader:       getAuthHeader(systemToken),
	}
}

func TestLogEventAndCUDs(t *testing.T) {
	testQName := appdef.NewQName("test", "Article")

	logCap := logger.StartCapture(t, logger.LogLevelVerbose)

	app := setUp(t, func(wsb appdef.IWorkspaceBuilder, cfg *istructsmem.AppConfigType) {
		wsb.AddCDoc(testQName).AddField("Name", appdef.DataKind_string, false)
		addTestCUDCommand(wsb, cfg)
		addStandardTestRoles(wsb)
	})
	defer tearDown(app)

	t.Run("verbose level emits event and cud log entries", func(t *testing.T) {
		require := require.New(t)

		logCap.Reset()

		// insert
		req := bus.Request{
			WSID:     1,
			AppQName: istructs.AppQName_untill_airs_bp,
			Resource: "c.sys.CUD",
			Body:     []byte(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.Article","Name":"hello"}}]}`),
			Header:   app.sysAuthHeader,
		}
		cmdRespMeta, cmdResp, sysErr := bus.GetCommandResponse(app.ctx, app.requestSender, req)
		require.NoError(sysErr)
		require.Equal(http.StatusOK, cmdRespMeta.StatusCode)
		newID := cmdResp.NewIDs["1"]
		require.NotZero(newID)

		logCap.HasLine("stage=cp.plog_saved", "evqname=sys.CUD")
		logCap.HasLine(fmt.Sprintf("rectype=%s", testQName), fmt.Sprintf("recid=%d", newID), "op=create")

		// update: newfields=world, oldfields=hello
		logCap.Reset()
		req.Body = fmt.Appendf(nil, `{"cuds":[{"sys.ID":%d,"fields":{"sys.QName":"test.Article","Name":"world"}}]}`, newID)
		cmdRespMeta, _, sysErr = bus.GetCommandResponse(app.ctx, app.requestSender, req)
		require.NoError(sysErr)
		require.Equal(http.StatusOK, cmdRespMeta.StatusCode)

		logCap.HasLine(
			fmt.Sprintf("recid=%d", newID),
			"op=update",
			fmt.Sprintf(`newfields={\"Name\":\"world\",\"sys.ID\":%d,\"sys.IsActive\":true,\"sys.QName\":\"test.Article\"}`, newID),
			fmt.Sprintf(`oldfields={\"Name\":\"hello\",\"sys.ID\":%d,\"sys.IsActive\":true,\"sys.QName\":\"test.Article\"}`, newID),
		)
	})

	t.Run("info level emits nothing", func(t *testing.T) {
		require := require.New(t)
		defer logger.SetLogLevelWithRestore(logger.LogLevelInfo)()

		logCap.Reset()

		req := bus.Request{
			WSID:     1,
			AppQName: istructs.AppQName_untill_airs_bp,
			Resource: "c.sys.CUD",
			Body:     []byte(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"test.Article"}}]}`),
			Header:   app.sysAuthHeader,
		}
		_, _, sysErr := bus.GetCommandResponse(app.ctx, app.requestSender, req)
		require.NoError(sysErr)
		logCap.NotContains("evqname")
	})
}

func TestSyncProjectorLogging(t *testing.T) {
	cudQName := appdef.NewQName(appdef.SysPackage, "CUD")
	projQName := appdef.NewQName(appdef.SysPackage, "TestProj")
	projExtension := "p." + projQName.String() // extension attr value used by sync actualizer

	t.Run("sp.triggeredby and sp.success on success", func(t *testing.T) {
		require := require.New(t)
		logCap := logger.StartCapture(t, logger.LogLevelVerbose)

		app := setUp(t, func(wsb appdef.IWorkspaceBuilder, cfg *istructsmem.AppConfigType) {
			addTestCUDRecords(wsb)
			addTestCUDCommand(wsb, cfg)
			addStandardTestRoles(wsb)
			cfg.AddSyncProjectors(istructs.Projector{
				Name: projQName,
				Func: func(istructs.IPLogEvent, istructs.IState, istructs.IIntents) error { return nil },
			})
			wsb.AddProjector(projQName).SetSync(true).Events().Add(
				[]appdef.OperationKind{appdef.OperationKind_Execute},
				filter.QNames(cudQName))
		})
		defer tearDown(app)

		// warm up: trigger partition recovery before the measured portion
		sendCUD_3Inserts(t, 1, app)

		logCap.Reset()
		sendCUD_3Inserts(t, 1, app)

		// per-projector logs: extension has "p." prefix; context attrs woffset/poffset/evqname are enriched
		logCap.HasLine("stage=sp.triggeredby", "extension="+projExtension, "woffset=", "poffset=", "evqname=")
		logCap.HasLine("stage=sp.success", "extension="+projExtension, "woffset=", "poffset=", "evqname=")
		// command-processor emits its own sp.success on sync projectors success — total must be 2
		// like this:
		// time=2026-03-25T11:41:09.726+03:00 level=DEBUG msg="" src=actualizers.newSyncBranch.func1:97 stage=sp.success extension=p.sys.TestProj woffset=3 poffset=4 evqname=sys.CUD
		// time=2026-03-25T11:41:09.726+03:00 level=DEBUG msg="" src=command.setUp.setUp.ProvideServiceFactory.func5.func6.2:82 stage=sp.success woffset=3 poffset=4 evqname=sys.CUD
		// note: extension attrib missing in the 1st line, actually it is added by router
		require.Equal(2, strings.Count(logCap.String(), "stage=sp.success"))

		// let's find the sp.success line emitted by the cmd proc, not by sync acrualizer
		cmdSuccessLine := ""
		for line := range strings.SplitSeq(logCap.String(), "\n") {
			if strings.Contains(line, "stage=sp.success") &&
				!strings.Contains(line, "extension="+projExtension) {
				cmdSuccessLine = line
				break
			}
		}
		// command-processor success line: must have context attrs and a different extension than projector
		require.NotEmpty(cmdSuccessLine)
		require.Contains(cmdSuccessLine, "stage=sp.success")
		require.Contains(cmdSuccessLine, "woffset=")
		require.Contains(cmdSuccessLine, "poffset=")
		require.Contains(cmdSuccessLine, "evqname=")

	})
}
