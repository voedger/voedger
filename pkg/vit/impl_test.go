/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

// Heeus integration test
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
	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func TestBasicUsage_SharedTestConfig(t *testing.T) {
	require := require.New(t)

	hit := NewHIT(t, &SharedConfig_Simple)
	ws := hit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("basic run", func(t *testing.T) {
		body := `{"args": {},"elements":[{"fields":["NumGoroutines"]}]}`
		resp := hit.PostWS(ws, "q.sys.GRCount", body)
		resp.Println()
	})

	t.Run("no Teardown() in previous test -> panic on quering HIT for the same shared config", func(t *testing.T) {
		require.Panics(func() { NewHIT(t, &SharedConfig_Simple) })
	})

	hit.TearDown()
	t.Run("query again the same shared config -> HIT with an existing HVM is returned", func(t *testing.T) {
		newHit := NewHIT(t, &SharedConfig_Simple)
		defer newHit.TearDown()
		body := `{"args": {},"elements":[{"fields":["NumGoroutines"]}]}`
		resp := newHit.PostWS(ws, "q.sys.GRCount", body)
		resp.Println()
		require.Equal(unsafe.Pointer(hit), unsafe.Pointer(newHit))
	})
}

func TestBasicUsage_WorkWithFunctions(t *testing.T) {
	require := require.New(t)
	hit := NewHIT(t, &SharedConfig_Simple)
	defer hit.TearDown()

	ws := hit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("query", func(t *testing.T) {
		body := `{"args": {"Text": "world"},"elements": [{"fields": ["Res"]}]}`
		resp := hit.PostWS(ws, "q.sys.Echo", body)
		require.Equal(`{"sections":[{"type":"","elements":[[[["world"]]]]}]}`, resp.Body)
		require.Equal("world", resp.SectionRow()[0])
		require.Equal("world", resp.Sections[0].Elements[0][0][0][0])
		require.Equal(http.StatusOK, resp.HTTPResp.StatusCode)
		require.Empty(resp.NewIDs)           // not used for queries
		require.Zero(resp.CurrentWLogOffset) // not used for queries
		resp.Println()                       // e.g. {"sections":[{"type":"","elements":[[[["world"]]]]}]}
	})

	t.Run("command", func(t *testing.T) {
		body := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"untill.air_table_plan","name":"test"}}]}`
		resp := hit.PostWS(ws, "c.sys.CUD", body)
		require.Len(resp.NewIDs, 1)
		require.True(resp.NewID() > 1)
		require.True(resp.CurrentWLogOffset > 0)
		require.Equal(http.StatusOK, resp.HTTPResp.StatusCode)
		require.Empty(resp.Sections)                 // not used for commands
		require.Panics(func() { resp.SectionRow() }) // panics if not a query
		resp.Println()                               // e.g. {"currentWLogOffset":15,"newIDs":{"1":322685000131073}}
	})
}

func TestBasicUsage_Workspaces(t *testing.T) {
	require := require.New(t)
	hit := NewHIT(t, &SharedConfig_Simple)
	defer hit.TearDown()

	t.Run("create workspace manually", func(t *testing.T) {
		// create and sign in an owner
		loginName := hit.NextName()
		login := hit.SignUp(loginName, "ownerpwd", istructs.AppQName_test1_app1)
		ownerPrincipal := hit.SignIn(login) // will wait for the user workspace here

		// create a workspace and wait for its initialization
		wsp := WSParams{
			Name:         "my workspace",
			TemplateName: "test_template",  // from SharedConfig_Simple
			InitDataJSON: `{"IntFld": 42}`, // intFld is required field, from SharedConfig_Simple
			Kind:         QNameTestWSKind,
			ClusterID:    istructs.MainClusterID,
		}
		newWS := hit.CreateWorkspace(wsp, ownerPrincipal)

		// use PostWS() to send requests to the workspace
		// authorized automatically by the workspace owner
		// non-200 response -> test failed
		body := `{"args": {},"elements":[{"fields":["NumGoroutines"]}]}`
		resp := hit.PostWS(newWS, "q.sys.GRCount", body)
		resp.Println()
	})

	t.Run("work with workspaces declared in the config", func(t *testing.T) {
		require.Panics(func() { hit.WS(istructs.AppQName_test2_app1, "test_ws") })
		require.Panics(func() { hit.WS(istructs.AppQName_test1_app1, "unknown") })

		ws := hit.WS(istructs.AppQName_test1_app1, "test_ws")

		body := `{"args": {},"elements":[{"fields":["NumGoroutines"]}]}`
		resp := hit.PostWS(ws, "q.sys.GRCount", body)
		resp.Println()
	})
}

func TestBasicUsage_N10N(t *testing.T) {
	require := require.New(t)
	hit := NewHIT(t, &SharedConfig_Simple)

	ws := hit.WS(istructs.AppQName_test1_app1, "test_ws")

	n10nChan := hit.SubscribeForN10n(ws, QNameTestView)

	// call test update to the view
	body := fmt.Sprintf(`
 		{
 			"App": "%s",
 			"Projection": "my.View",
 			"WS": %d
 		}`, istructs.AppQName_test1_app1.String(), ws.WSID)
	hit.Post("n10n/update/13", body)

	offset := <-n10nChan
	log.Println(offset)

	// the cannel will be closed automatically on hit.TearDown()
	hit.TearDown()
	_, ok := <-n10nChan
	require.False(ok)
}

func TestBasicUsage_POST(t *testing.T) {
	require := require.New(t)
	hit := NewHIT(t, &SharedConfig_Simple)
	defer hit.TearDown()

	// 200ok is expected by default
	// unexpected result code -> test is failed
	// response body is read out and closed
	bodyEcho := `{"args": {"Text": "world"},"elements": [{"fields": ["Res"]}]}`
	bodyCUD := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"untill.air_table_plan","name":"test"}}]}`
	httpResp := hit.Post("api/test1/app1/1/q.sys.Echo", bodyEcho) // HTTPResponse is returned
	require.Equal(`{"sections":[{"type":"","elements":[[[["world"]]]]}]}`, httpResp.Body)

	ws := hit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("low-level POST with authorization by token", func(t *testing.T) {
		hit.Post(fmt.Sprintf("api/test1/app1/%d/c.sys.CUD", ws.WSID), bodyCUD, utils.Expect403())
		httpResp = hit.Post(fmt.Sprintf("api/test1/app1/%d/c.sys.CUD", ws.WSID), bodyCUD, utils.WithAuthorizeBy(ws.Owner.Token))
		httpResp.Println()
	})

	t.Run("low-level POST with authorization by header", func(t *testing.T) {
		httpResp = hit.Post(fmt.Sprintf("api/test1/app1/%d/c.sys.CUD", ws.WSID), bodyCUD, utils.WithHeaders(coreutils.Authorization, "Bearer "+ws.Owner.Token))
		httpResp.Println()
	})

	t.Run("headers and cookies", func(t *testing.T) {
		hit.PostWS(ws, "q.sys.Echo", bodyEcho,
			utils.WithHeaders("Test-header", "Test header value"),
			utils.WithCookies("Test-cookie", "test cookie value"),
		)
	})

	t.Run("app-level POST with authorization", func(t *testing.T) {
		hit.PostApp(istructs.AppQName_test1_app1, ws.WSID, "c.sys.CUD", bodyCUD, utils.Expect403())
		resp := hit.PostApp(istructs.AppQName_test1_app1, ws.WSID, "c.sys.CUD", bodyCUD, utils.WithAuthorizeBy(ws.Owner.Token)) // FuncResponse is returned
		require.True(resp.NewID() > 0)
		require.True(resp.CurrentWLogOffset > 0)
		require.Empty(resp.Sections)                 // not used for commands
		require.Panics(func() { resp.SectionRow() }) // not used for commands
	})

	t.Run("custom response handler", func(t *testing.T) {
		resp := hit.PostWS(ws, "q.sys.Echo", bodyEcho, utils.WithResponseHandler(func(httpResp *http.Response) {
			bytes, err := io.ReadAll(httpResp.Body)
			require.Nil(err, err)
			log.Println(string(bytes))

			// response body must be explicitly closed
			httpResp.Body.Close()
		}))

		// custom response handler used -> Body is not saved
		require.Empty(resp.Body)
	})
}

func TestBasicUsage_OwnTestConfig(t *testing.T) {
	ownCfg := NewOwnHITConfig(WithApp(istructs.AppQName_test1_app1, EmptyApp))

	t.Run("basic - HIT on own config", func(t *testing.T) {
		hit := NewHIT(t, &ownCfg)
		defer hit.TearDown()

		body := `
			{
				"args": {},
				"elements": [
					{
						"fields":["NumGoroutines"]
					}
				]
			}`
		resp := hit.PostApp(istructs.AppQName_test1_app1, 1, "q.sys.GRCount", body)
		resp.Println()
	})
}

func TestEmailExpectation(t *testing.T) {
	require := require.New(t)
	hit := NewHIT(t, &SharedConfig_Simple)
	defer hit.TearDown()

	// provide HIT email sending chan to the IBundledHostState, then use it to send an email
	s := state.ProvideAsyncActualizerStateFactory()(context.Background(), &nilAppStructs{}, nil, nil, nil, nil, 1, 0,
		state.WithEmailMessagesChan(hit.emailMessagesChan))
	k, err := s.KeyBuilder(state.SendMailStorage, appdef.NullQName)
	require.NoError(err)

	// construct the email
	k.PutInt32(state.Field_Port, 1)
	k.PutString(state.Field_Host, "localhost")
	k.PutString(state.Field_Username, "user")
	k.PutString(state.Field_Password, "pwd")
	k.PutString(state.Field_Subject, "Greeting")
	k.PutString(state.Field_From, "from@email.com")
	k.PutString(state.Field_To, "to0@email.com")
	k.PutString(state.Field_To, "to1@email.com")
	k.PutString(state.Field_CC, "cc0@email.com")
	k.PutString(state.Field_CC, "cc1@email.com")
	k.PutString(state.Field_BCC, "bcc0@email.com")
	k.PutString(state.Field_BCC, "bcc1@email.com")
	k.PutString(state.Field_Body, "Hello world")

	t.Run("basic usage", func(t *testing.T) {
		emailCaptor := hit.ExpectEmail()
		require.Nil(s.NewValue(k))
		readyToFlush, err := s.ApplyIntents()
		require.True(readyToFlush)
		require.NoError(err)
		require.NoError(s.FlushBundles())
		email := emailCaptor.Capture()
		log.Println(email)
	})

	t.Run("fail the test if an unexpected email is sent", func(t *testing.T) {
		hit.TearDown()
		newT := &testing.T{}
		hit = NewHIT(newT, &SharedConfig_Simple)
		require.Nil(s.NewValue(k))
		readyToFlush, err := s.ApplyIntents()
		require.True(readyToFlush)
		require.NoError(err)
		require.NoError(s.FlushBundles())
		hit.TearDown()
		require.True(newT.Failed())
		hit = NewHIT(t, &SharedConfig_Simple)
	})

	hit.TearDown()
}

type nilAppStructs struct {
	istructs.IAppStructs
}

func (s *nilAppStructs) Events() istructs.IEvents           { return nil }
func (s *nilAppStructs) Records() istructs.IRecords         { return nil }
func (s *nilAppStructs) ViewRecords() istructs.IViewRecords { return nil }
func (s *nilAppStructs) AppDef() appdef.IAppDef             { return nil }
