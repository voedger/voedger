/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/bbolt"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
	it "github.com/voedger/voedger/pkg/vit"
	"github.com/voedger/voedger/pkg/vvm"
)

type rr struct {
	istructs.NullObject
	res string
}

func (r *rr) AsString(string) string {
	return r.res
}

func TestBug_QueryProcessorMustStopOnClientDisconnect(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	require := require.New(t)
	clientDisconnected := make(chan interface{})
	expectedErrors := make(chan error)
	it.MockQryExec = func(input string, _ istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		rr := &rr{res: input}
		require.NoError(callback(rr))
		<-clientDisconnected // what for http client to receive the first element and disconnect
		// now wait for error context.Cancelled. It will be occurred immediately because an async pipeline works within queryprocessor
		for err == nil {
			err = callback(rr)
		}
		expectedErrors <- err
		return err
	}
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	// send POST request
	body := `{"args": {"Input": "world"},"elements": [{"fields": ["Res"]}]}`
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	vit.PostWS(ws, "q.app1pkg.MockQry", body, httpu.WithResponseHandler(func(httpResp *http.Response) {
		// read out the first part of the respoce (the serer will not send the next one before writing something in goOn)
		entireResp := []byte{}
		var err error
		n := 0
		for string(entireResp) != `{"sections":[{"type":"","elements":[[[["world"]]]` {
			if n == 0 && errors.Is(err, io.EOF) {
				t.Fatal()
			}
			buf := make([]byte, 512)
			n, err = httpResp.Body.Read(buf)
			entireResp = append(entireResp, buf[:n]...)
		}

		// break the connection during request handling
		httpResp.Request.Body.Close()
		httpResp.Body.Close()
		close(clientDisconnected) // the func will start to send the second part. That will be failed because the request context is closed
	}))

	require.ErrorIs(<-expectedErrors, context.Canceled)
	// expecting that there are no additional errors: nothing hung, queryprocessor is done, router does not try to write to a closed connection etc
}

func Test409OnRepeatedlyUsedRawIDsInResultCUDs_(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	it.MockCmdExec = func(_ string, args istructs.ExecCommandArgs) error {
		// the same rawID 2 times -> 500 internal server error
		kb, err := args.State.KeyBuilder(sys.Storage_Record, it.QNameApp1_CDocCategory)
		if err != nil {
			return err
		}
		sv, err := args.Intents.NewValue(kb)
		if err != nil {
			return err
		}
		sv.PutRecordID(appdef.SystemField_ID, 1)

		kb, err = args.State.KeyBuilder(sys.Storage_Record, it.QNameApp1_CDocCategory)
		if err != nil {
			return err
		}
		sv, err = args.Intents.NewValue(kb)
		if err != nil {
			return err
		}
		sv.PutRecordID(appdef.SystemField_ID, 1)
		return nil
	}
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	vit.PostWS(ws, "c.app1pkg.MockCmd", `{"args":{"Input":"Str"}}`, httpu.Expect409()).Println()
}

// https://github.com/voedger/voedger/issues/2759
func TestWrongFieldReferencedByRefField(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	body := `{"cuds":[{"fields":{"sys.ID": 1,"sys.QName":"app1pkg.cdoc1"}}]}`
	cdoc1ID := vit.PostWS(ws, "c.sys.CUD", body).NewID()

	body = fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID": 1,"sys.QName":"app1pkg.cdoc2","field2": %d}}]}`, cdoc1ID)
	vit.PostWS(ws, "c.sys.CUD", body).NewID()

	body = `{"args":{"Schema":"app1pkg.cdoc2"},"elements":[{"fields": ["field2","sys.ID"], "refs":[["field2","unexistingFieldInTargetDoc"]]}]}`
	vit.PostWS(ws, "q.sys.Collection", body, it.Expect400("ref field field2 references to table app1pkg.cdoc1 that does not contain field unexistingFieldInTargetDoc"))
}

// https://github.com/voedger/voedger/issues/3046
func TestWSNameCausesMaxPseudoWSID(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	// BaseWSID for entity "2062880497" is 65535 so due of comparing <istructs.MaxPseudoBaseWSID in router the WSID
	// considered as not pseudo -> panic on try to call AsQName() on a missing workspace descriptor on a non-inited ws
	pseudoWSID := coreutils.GetPseudoWSID(istructs.NullWSID, "2062880497", istructs.CurrentClusterID())
	url := fmt.Sprintf("api/v2/apps/sys/registry/workspaces/%d/commands/registry.CreateLogin", pseudoWSID)
	body := fmt.Sprintf(`{"args":{"Login":"2062880497","AppName":"%s","SubjectKind":%d,"WSKindInitializationData":"{}","ProfileCluster":%d},"unloggedArgs":{"Password":"1"}}`,
		istructs.AppQName_test1_app1, istructs.SubjectKind_Device, istructs.CurrentClusterID())
	resp := vit.Func(url, body, httpu.WithMethod(http.MethodPost))
	m := map[string]interface{}{}
	require.NoError(vit.T, json.Unmarshal([]byte(resp.Body), &m))
	deviceLogin := it.NewLogin("2062880497", "1", istructs.AppQName_test1_app1, istructs.SubjectKind_Device,
		istructs.CurrentClusterID())

	// need to wait for init anyway, otherwise vit.Time.Add(1 day) on next test -> token expired during the device ws init
	vit.SignIn(deviceLogin)
}

// AIR-4355: events read sequentially from the log (ReadToTheEnd) and retained after the read
// transaction closed must own their bytes. The storage read callback hands the event a zero-copy
// view that, with a bbolt-backed storage, aliases mmap pages unmapped once the file grows and
// re-maps - reading a retained event afterwards yields garbage or SIGSEGV.
func TestBug_BatchedLogEventsMustOwnTheirBytes(t *testing.T) {
	require := require.New(t)

	cfg := getBboltVITCfg(t)
	vit := it.NewVIT(t, &cfg)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	airTablePlan := appdef.NewQName("app1pkg", "air_table_plan")

	const eventsCnt = 16
	expected := make(map[string]bool, eventsCnt)
	for i := range eventsCnt {
		name := fmt.Sprintf("buyer-%d", i)
		expected[name] = true
		body := fmt.Sprintf(`{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"%s","name":"%s"}}]}`, airTablePlan, name)
		vit.PostWS(ws, "c.sys.CUD", body)
	}

	// sequentially read the whole WLog and retain the returned events
	as, err := vit.BuiltIn(istructs.AppQName_test1_app1)
	require.NoError(err)

	var events []istructs.IWLogEvent
	require.NoError(as.Events().ReadWLog(context.Background(), ws.WSID, istructs.FirstOffset, istructs.ReadToTheEnd,
		func(_ istructs.Offset, event istructs.IWLogEvent) error {
			events = append(events, event)
			return nil
		}))

	// churn the very same bbolt storage the events were read from (public storage API only) so the
	// file grows and re-maps, unmapping the pages the retained events may still point into
	storage, err := vit.IAppStorageProvider.AppStorage(istructs.AppQName_test1_app1)
	require.NoError(err)

	small := bytes.Repeat([]byte{0xCD}, 96)
	smallBatch := make([]istorage.BatchItem, 0, 8192)
	for i := range 8192 {
		smallBatch = append(smallBatch, istorage.BatchItem{
			PKey: []byte("AIR4355-churn-small"), CCols: fmt.Appendf(nil, "%010d", i), Value: small,
		})
	}
	require.NoError(storage.PutBatch(smallBatch))

	large := bytes.Repeat([]byte{0xAB}, int(appdef.MaxFieldLength))
	for round := range 2 {
		largeBatch := make([]istorage.BatchItem, 0, 800)
		for i := range 800 {
			largeBatch = append(largeBatch, istorage.BatchItem{
				PKey: fmt.Appendf(nil, "AIR4355-churn-large-%d", round), CCols: fmt.Appendf(nil, "%010d", i), Value: large,
			})
		}
		require.NoError(storage.PutBatch(largeBatch))
	}

	// the retained events must still expose their original field values; with the bug they read
	// freed/re-mapped memory here (mismatch or SIGSEGV)
	found := make(map[string]bool, eventsCnt)
	for _, event := range events {
		event.CUDs(func(rec istructs.ICUDRow) bool {
			if rec.QName() == airTablePlan {
				found[rec.AsString("name")] = true
			}
			return true
		})
		event.Release()
	}
	for name := range expected {
		require.True(found[name], name)
	}
}

func getBboltVITCfg(t *testing.T) it.VITConfig {
	dbDir, err := os.MkdirTemp("", "AIR-4355") //nolint:usetesting // bbolt driver does not close the connection so t.TempDir() fails
	require.NoError(t, err)

	return it.NewOwnVITConfig(
		it.WithApp(istructs.AppQName_test1_app1, it.ProvideApp1,
			it.WithUserLogin("login", "pwd"),
			it.WithChildWorkspace(it.QNameApp1_TestWSKind, "test_ws", "", "", "login", map[string]interface{}{"IntFld": 42}),
		),
		it.WithVVMConfig(func(cfg *vvm.VVMConfig) {
			cfg.StorageFactory = func(time timeu.ITime) (istorage.IAppStorageFactory, error) {
				return bbolt.Provide(bbolt.ParamsType{DBDir: dbDir}, time), nil
			}
		}),
	)
}
