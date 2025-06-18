/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

// Voedger integration test
package vit

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	"github.com/voedger/voedger/pkg/state/stateprovide"
	"github.com/voedger/voedger/pkg/sys"
)

func TestBasicUsage_SharedTestConfig(t *testing.T) {
	require := require.New(t)

	vit := NewVIT(t, &SharedConfig_App1)
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("basic run", func(t *testing.T) {
		body := `{"args": {},"elements":[{"fields":["NumGoroutines"]}]}`
		resp := vit.PostWS(ws, "q.sys.GRCount", body)
		resp.Println()
	})

	t.Run("no Teardown() in previous test -> panic on quering VIT for the same shared config", func(t *testing.T) {
		require.Panics(func() { NewVIT(t, &SharedConfig_App1) })
	})

	vit.TearDown()
	t.Run("query again the same shared config -> VIT with an existing VVM is returned", func(t *testing.T) {
		newVit := NewVIT(t, &SharedConfig_App1)
		defer newVit.TearDown()
		body := `{"args": {},"elements":[{"fields":["NumGoroutines"]}]}`
		resp := newVit.PostWS(ws, "q.sys.GRCount", body)
		resp.Println()
		require.Equal(unsafe.Pointer(vit), unsafe.Pointer(newVit))
	})
}

func TestBasicUsage_WorkWithFunctions(t *testing.T) {
	require := require.New(t)
	vit := NewVIT(t, &SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("query", func(t *testing.T) {
		body := `{"args": {"Text": "world"},"elements": [{"fields": ["Res"]}]}`
		resp := vit.PostWS(ws, "q.sys.Echo", body)
		require.Equal(`{"sections":[{"type":"","elements":[[[["world"]]]]}]}`, resp.Body)
		require.Equal("world", resp.SectionRow()[0])
		require.Equal("world", resp.Sections[0].Elements[0][0][0][0])
		require.Equal(http.StatusOK, resp.HTTPResp.StatusCode)
		require.Empty(resp.NewIDs)           // not used for queries
		require.Zero(resp.CurrentWLogOffset) // not used for queries
		resp.Println()                       // e.g. {"sections":[{"type":"","elements":[[[["world"]]]]}]}
	})

	t.Run("command", func(t *testing.T) {
		body := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.air_table_plan","name":"test"}}]}`
		resp := vit.PostWS(ws, "c.sys.CUD", body)
		require.Len(resp.NewIDs, 1)
		require.Greater(resp.NewID(), istructs.RecordID(1))
		require.Greater(resp.CurrentWLogOffset, istructs.Offset(0))
		require.Equal(http.StatusOK, resp.HTTPResp.StatusCode)
		require.Empty(resp.Sections)                 // not used for commands
		require.Panics(func() { resp.SectionRow() }) // panics if not a query
		resp.Println()                               // e.g. {"currentWLogOffset":15,"newIDs":{"1":322685000131073}}
	})
}

func TestBasicUsage_Workspaces(t *testing.T) {
	require := require.New(t)
	vit := NewVIT(t, &SharedConfig_App1)
	defer vit.TearDown()

	t.Run("create workspace manually", func(t *testing.T) {
		// create and sign in an owner
		loginName := vit.NextName()
		login := vit.SignUp(loginName, "ownerpwd", istructs.AppQName_test1_app1)
		ownerPrincipal := vit.SignIn(login) // will wait for the user workspace here

		// create a workspace and wait for its initialization
		wsp := WSParams{
			Name:         "my workspace",
			TemplateName: "test_template",  // from SharedConfig_Simple
			InitDataJSON: `{"IntFld": 42}`, // intFld is required field, from SharedConfig_Simple
			Kind:         QNameApp1_TestWSKind,
			ClusterID:    istructs.CurrentClusterID(),
		}
		newWS := vit.CreateWorkspace(wsp, ownerPrincipal)

		// use PostWS() to send requests to the workspace
		// authorized automatically by the workspace owner
		// non-200 response -> test failed
		body := `{"args": {},"elements":[{"fields":["NumGoroutines"]}]}`
		resp := vit.PostWS(newWS, "q.sys.GRCount", body)
		resp.Println()
	})

	t.Run("work with workspaces declared in the config", func(t *testing.T) {
		require.Panics(func() { vit.WS(istructs.AppQName_test2_app1, "test_ws") })
		require.Panics(func() { vit.WS(istructs.AppQName_test1_app1, "unknown") })

		ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

		body := `{"args": {},"elements":[{"fields":["NumGoroutines"]}]}`
		resp := vit.PostWS(ws, "q.sys.GRCount", body)
		resp.Println()
	})
}

func TestBasicUsage_N10N(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	require := require.New(t)
	vit := NewVIT(t, &SharedConfig_App1)

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	n10nChan := vit.SubscribeForN10n(ws, QNameTestView)

	// call test update to the view
	vit.N10NUpdate(in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName(app1PkgName, "View"),
		WS:         ws.WSID,
	}, 13)

	offset := <-n10nChan
	log.Println(offset)

	// the cannel will be closed automatically on vit.TearDown()
	vit.TearDown()
	_, ok := <-n10nChan
	require.False(ok)
}

func TestBasicUsage_POST(t *testing.T) {
	require := require.New(t)
	vit := NewVIT(t, &SharedConfig_App1)
	defer vit.TearDown()

	// 200ok is expected by default
	// unexpected result code -> test is failed
	// response body is read out and closed
	bodyEcho := `{"args": {"Text": "world"},"elements": [{"fields": ["Res"]}]}`
	bodyCUD := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.air_table_plan","name":"test"}}]}`
	httpResp := vit.Func("api/test1/app1/1/q.sys.Echo", bodyEcho) // HTTPResponse is returned
	require.Equal(`{"sections":[{"type":"","elements":[[[["world"]]]]}]}`, httpResp.Body)

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("low-level POST with authorization by token", func(t *testing.T) {
		vit.Func(fmt.Sprintf("api/test1/app1/%d/c.sys.CUD", ws.WSID), bodyCUD, coreutils.Expect403())
		httpResp = vit.Func(fmt.Sprintf("api/test1/app1/%d/c.sys.CUD", ws.WSID), bodyCUD, coreutils.WithAuthorizeBy(ws.Owner.Token))
		httpResp.Println()
	})

	t.Run("low-level POST with authorization by header", func(t *testing.T) {
		httpResp = vit.Func(fmt.Sprintf("api/test1/app1/%d/c.sys.CUD", ws.WSID), bodyCUD, coreutils.WithHeaders(coreutils.Authorization, "Bearer "+ws.Owner.Token))
		httpResp.Println()
	})

	t.Run("headers and cookies", func(t *testing.T) {
		vit.PostWS(ws, "q.sys.Echo", bodyEcho,
			coreutils.WithHeaders("Test-header", "Test header value"),
			coreutils.WithCookies("Test-cookie", "test cookie value"),
		)
	})

	t.Run("app-level POST with authorization", func(t *testing.T) {
		vit.PostApp(istructs.AppQName_test1_app1, ws.WSID, "c.sys.CUD", bodyCUD, coreutils.Expect403())
		resp := vit.PostApp(istructs.AppQName_test1_app1, ws.WSID, "c.sys.CUD", bodyCUD, coreutils.WithAuthorizeBy(ws.Owner.Token)) // FuncResponse is returned
		require.Greater(resp.NewID(), istructs.NullRecordID)
		require.Greater(resp.CurrentWLogOffset, istructs.Offset(0))
		require.Empty(resp.Sections)                 // not used for commands
		require.Panics(func() { resp.SectionRow() }) // not used for commands
	})

	t.Run("custom response handler", func(t *testing.T) {
		resp := vit.PostWS(ws, "q.sys.Echo", bodyEcho, coreutils.WithResponseHandler(func(httpResp *http.Response) {
			bytes, err := io.ReadAll(httpResp.Body)
			require.NoError(err)
			log.Println(string(bytes))

			// response body must be explicitly closed
			httpResp.Body.Close()
		}))

		// custom response handler used -> Body is not saved
		require.Empty(resp.Body)
	})
}

func TestBasicUsage_OwnTestConfig(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ownCfg := NewOwnVITConfig(WithApp(istructs.AppQName_test1_app1, ProvideApp1))

	t.Run("basic - VIT on own config", func(t *testing.T) {
		vit := NewVIT(t, &ownCfg)
		defer vit.TearDown()

		body := `
			{
				"args": {},
				"elements": [
					{
						"fields":["NumGoroutines"]
					}
				]
			}`
		resp := vit.PostApp(istructs.AppQName_test1_app1, 1, "q.sys.GRCount", body)
		resp.Println()
	})
}

func TestEmailExpectation(t *testing.T) {
	require := require.New(t)
	vit := NewVIT(t, &SharedConfig_App1)
	defer vit.TearDown()

	// provide VIT email sending chan to the IBundledHostState, then use it to send an email
	s := stateprovide.ProvideAsyncActualizerStateFactory()(context.Background(), func() istructs.IAppStructs { return &nilAppStructs{} }, nil, nil, nil, nil, nil, nil, nil, 1, 0,
		state.WithEmailSenderOverride(vit.emailCaptor))
	k, err := s.KeyBuilder(sys.Storage_SendMail, appdef.NullQName)
	require.NoError(err)

	// construct the email
	k.PutInt32(sys.Storage_SendMail_Field_Port, 1)
	k.PutString(sys.Storage_SendMail_Field_Host, "localhost")
	k.PutString(sys.Storage_SendMail_Field_Username, "user")
	k.PutString(sys.Storage_SendMail_Field_Password, "pwd")
	k.PutString(sys.Storage_SendMail_Field_Subject, "Greeting")
	k.PutString(sys.Storage_SendMail_Field_From, "from@email.com")
	k.PutString(sys.Storage_SendMail_Field_To, "to0@email.com")
	k.PutString(sys.Storage_SendMail_Field_To, "to1@email.com")
	k.PutString(sys.Storage_SendMail_Field_CC, "cc0@email.com")
	k.PutString(sys.Storage_SendMail_Field_CC, "cc1@email.com")
	k.PutString(sys.Storage_SendMail_Field_BCC, "bcc0@email.com")
	k.PutString(sys.Storage_SendMail_Field_BCC, "bcc1@email.com")
	k.PutString(sys.Storage_SendMail_Field_Body, "Hello world")

	t.Run("basic usage", func(t *testing.T) {
		require.NotNil(s.NewValue(k))
		readyToFlush, err := s.ApplyIntents()
		require.True(readyToFlush)
		require.NoError(err)
		require.NoError(s.FlushBundles())
		email := vit.CaptureEmail()
		log.Println(email)
	})

	t.Run("fail the test if an unexpected email is sent", func(t *testing.T) {
		vit.TearDown()
		newT := &testing.T{}
		vit = NewVIT(newT, &SharedConfig_App1)
		require.NotNil(s.NewValue(k))
		readyToFlush, err := s.ApplyIntents()
		require.True(readyToFlush)
		require.NoError(err)
		require.NoError(s.FlushBundles())
		vit.TearDown()
		require.True(newT.Failed())
		vit = NewVIT(t, &SharedConfig_App1)
	})

	vit.TearDown()
}

func TestWithChildWorkspaceOfWorkspace(t *testing.T) {
	vit := NewVIT(t, &SharedConfig_App1)
	defer vit.TearDown()
	ws2 := vit.WS(istructs.AppQName_test1_app1, "test_ws2")
	body := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.options"}}]}`
	prnTestEmail := vit.GetPrincipal(istructs.AppQName_test1_app1, TestEmail)
	// allowed for login "123@123.com" despite he is not an owner of test_ws2
	vit.PostWS(ws2, "c.sys.CUD", body, coreutils.WithAuthorizeBy(prnTestEmail.Token))
}

type nilAppStructs struct {
	istructs.IAppStructs
}

func (s *nilAppStructs) Events() istructs.IEvents           { return nil }
func (s *nilAppStructs) Records() istructs.IRecords         { return nil }
func (s *nilAppStructs) ViewRecords() istructs.IViewRecords { return nil }
func (s *nilAppStructs) AppDef() appdef.IAppDef             { return nil }
