/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
	it "github.com/voedger/voedger/pkg/vit"
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

	// sned POST request
	body := `{"args": {"Input": "world"},"elements": [{"fields": ["Res"]}]}`
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	vit.PostWS(ws, "q.app1pkg.MockQry", body, coreutils.WithResponseHandler(func(httpResp *http.Response) {
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
	vit.PostWS(ws, "c.app1pkg.MockCmd", `{"args":{"Input":"Str"}}`, coreutils.Expect409()).Println()
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
	vit.PostWS(ws, "q.sys.Collection", body, coreutils.Expect400("ref field field2 references to table app1pkg.cdoc1 that does not contain field unexistingFieldInTargetDoc"))
}

// https://github.com/voedger/voedger/issues/3046
func TestWSNameCausesMaxPseudoWSID(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	// BaseWSID for entity "2062880497" is 65535 so due of comparing <istructs.MaxPseudoBaseWSID in router the WSID
	// considered as not pseudo -> panic on try to call AsQName() on a missing workspace descriptor on a non-inited ws
	deviceLogin := vit.SignUpDevice("2062880497", "1", istructs.AppQName_test1_app1)

	// need to wait for init anyway, otherwise vit.Time.Add(1 day) on next test -> token expired during the device ws init
	vit.SignIn(deviceLogin)
}
