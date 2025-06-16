/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/istructs"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_n10n_APIv1(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	testProjectionKey := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("app1pkg", "CategoryIdx"),
		WS:         ws.WSID,
	}

	offsetsChan, unsubscribe := vit.SubscribeForN10nUnsubscribe(testProjectionKey)

	// force projection update
	body := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"Awesome food"}}]}`
	resultOffsetOfCUD := vit.PostWS(ws, "c.sys.CUD", body).CurrentWLogOffset
	waitForOffset(t, resultOffsetOfCUD, offsetsChan)
	unsubscribe()

	_, offsetsChanOpened := <-offsetsChan
	require.False(t, offsetsChanOpened)
}

// [~server.n10n/it.CreateChannelSubscribeAndWatch~impl]
func TestBasicUsage_n10n_APIv2(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	// owning does not matter for notifications, need just a valid token
	token := ws.Owner.Token

	// subscribe
	body := fmt.Sprintf(`{
		"subscriptions": [
			{
				"entity":"app1pkg.CategoryIdx",
				"wsid": %[1]d
			},
			{
				"entity":"app1pkg.DailyIdx",
				"wsid": %[1]d
			}
		],
		"expiresIn": 42
	}`, ws.WSID)
	resp := vit.POST("api/v2/apps/test1/app1/notifications", body,
		coreutils.WithAuthorizeBy(token),
		coreutils.WithLongPolling(),
	)

	offsetsChan, channelID, waitForDone := federation.ListenSSEEvents(resp.HTTPResp.Request.Context(), resp.HTTPResp.Body)

	// force projections update
	body = `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"Awesome food"}}]}`
	resultOffsetOfCategoryCUD := vit.PostWS(ws, "c.sys.CUD", body).CurrentWLogOffset
	body = `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.Daily","Year":42}}]}`
	resultOffsetOfDailyCUD := vit.PostWS(ws, "c.sys.CUD", body).CurrentWLogOffset

	// read events
	waitForOffset(t, resultOffsetOfCategoryCUD, offsetsChan)
	waitForOffset(t, resultOffsetOfDailyCUD, offsetsChan)

	// unsubscribe
	// [~server.n10n/it.Unsubscribe~impl]
	url := fmt.Sprintf("api/v2/apps/test1/app1/notifications/%s/workspaces/%d/subscriptions/app1pkg.CategoryIdx", channelID, ws.WSID)
	vit.POST(url, "",
		coreutils.WithMethod(http.MethodDelete),
		coreutils.WithAuthorizeBy(token),
		coreutils.Expect204(),
	)
	url = fmt.Sprintf("api/v2/apps/test1/app1/notifications/%s/workspaces/%d/subscriptions/app1pkg.DailyIdx", channelID, ws.WSID)
	vit.POST(url, "",
		coreutils.WithMethod(http.MethodDelete),
		coreutils.WithAuthorizeBy(token),
		coreutils.Expect204(),
	)

	// force updates again to check that no new notifications arrived after unsubscribe
	body = `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"Awesome food"}}]}`
	vit.PostWS(ws, "c.sys.CUD", body)
	body = `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.Daily","Year":42}}]}`
	vit.PostWS(ws, "c.sys.CUD", body)

	// close the initial connection
	// SSE listener channel should be closed after that
	resp.HTTPResp.Body.Close()

	for range offsetsChan {
	}
	waitForDone()
}

func waitForOffset(t *testing.T, expectedOffset istructs.Offset, offsetCh federation.OffsetsChan) {
	start := time.Now()
	for actualOffset := range offsetCh {
		if actualOffset == expectedOffset {
			return
		}
		if time.Since(start) > 10*time.Second {
			t.Fatal()
		}
	}
}

func TestChannelExpiration_V2(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	// owning does not matter for notifications, need just a valid token
	token := ws.Owner.Token

	// subscribe
	body := fmt.Sprintf(`{
		"subscriptions": [
			{
				"entity":"app1pkg.CategoryIdx",
				"wsid": %[1]d
			}
		],
		"expiresIn": 3
	}`, ws.WSID)
	resp := vit.POST("api/v2/apps/test1/app1/notifications", body,
		coreutils.WithAuthorizeBy(token),
		coreutils.WithLongPolling(),
	)

	offsetsChan, _, waitForDone := federation.ListenSSEEvents(resp.HTTPResp.Request.Context(), resp.HTTPResp.Body)

	// make the channel expire
	testingu.MockTime.Add(4 * time.Second)

	// force update the expired channel
	body = `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"Awesome food"}}]}`
	vit.PostWS(ws, "c.sys.CUD", body)

	// expect SSE listener is finished
	for range offsetsChan {
	}
	waitForDone()
}

func TestN10NSubscribeErrors(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("401 unauthorized", func(t *testing.T) {
		t.Run("no token", func(t *testing.T) {
			vit.POST("api/v2/apps/test1/app1/notifications", "{}", coreutils.Expect401()).Println()
		})

		t.Run("expired token", func(t *testing.T) {
			testingu.MockTime.Add(24 * time.Hour)
			vit.POST("api/v2/apps/test1/app1/notifications", "{}",
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.Expect401(),
			).Println()
			vit.RefreshTokens()
		})
	})

	t.Run("bad requests", func(t *testing.T) {
		cases := []struct {
			body     string
			expected string
		}{
			{"", "failed to unmarshal request body"},
			{"{}", "no subscriptions provided"},
			{`{"subscriptions":[]}`, "no subscriptions provided"},
			{`{"subscriptions":42}`, "cannot unmarshal number into Go struct field"},
			{`{"wrong":42}`, `unknown field "wrong"`},
			{`{"subscriptions":[{"wrong":42}]}`, `unknown field "wrong"`},
			{`{"subscriptions":[{"entity":"test.test"}]}`, `entity and\or wsid is not provided`},
			{`{"subscriptions":[{"wsid":42}]}`, `entity and\or wsid is not provided`},
			{`{"subscriptions":[{"entity":42,"wsid":42}]}`, `cannot unmarshal number into Go struct`},
			{`{"subscriptions":[{"entity":42,"wsid":"str"}]}`, `trying to unmarshal "\"str\"" into Number`},
			{`{"subscriptions":[{"entity":"test.test","wsid":42}],"expiresIn":"str"}`, `cannot unmarshal string into Go struct`},
			{`{"subscriptions":[{"entity":"wrong","wsid":42}]}`, `failed to parse entity wrong as a QName`},
			{`{"subscriptions":[{"entity":"test.test","wsid":-1}]}`, `number overflow: -1 to WSID`},
			{`{"subscriptions":[{"entity":"test.test","wsid":42}],"expiresIn":-1}`, `invalid expiresIn value -1`},
		}
		for _, c := range cases {
			t.Run(c.body, func(t *testing.T) {
				vit.POST("api/v2/apps/test1/app1/notifications", c.body,
					coreutils.WithAuthorizeBy(ws.Owner.Token),
					coreutils.Expect400(c.expected),
				).Println()
			})
		}
	})
}

func TestN10NSubscribeToExtraErrors(t *testing.T) {
	testSubscriptionErrors(t, http.MethodPut)
}

func TestN10NUnsubscribeErrors(t *testing.T) {
	testSubscriptionErrors(t, http.MethodDelete)
}

func testSubscriptionErrors(t *testing.T, method string) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	// owning does not matter for notifications, need just a valid token
	token := ws.Owner.Token

	// subscribe
	body := fmt.Sprintf(`{"subscriptions": [{"entity":"app1pkg.CategoryIdx","wsid": %d}],"expiresIn": 42}`, ws.WSID)
	resp := vit.POST("api/v2/apps/test1/app1/notifications", body,
		coreutils.WithAuthorizeBy(token),
		coreutils.WithLongPolling(),
	)
	offsetsChan, channelID, waitForDone := federation.ListenSSEEvents(resp.HTTPResp.Request.Context(), resp.HTTPResp.Body)
	url := fmt.Sprintf("api/v2/apps/test1/app1/notifications/%s/workspaces/%d/subscriptions/app1pkg.CategoryIdx", channelID, ws.WSID)

	t.Run("401", func(t *testing.T) {
		t.Run("no token", func(t *testing.T) {
			vit.POST(url, "", coreutils.WithMethod(method), coreutils.Expect401())
		})

		t.Run("expired token", func(t *testing.T) {
			testingu.MockTime.Add(24 * time.Hour)
			vit.POST(url, "",
				coreutils.WithAuthorizeBy(ws.Owner.Token),
				coreutils.WithMethod(method),
				coreutils.Expect401(),
			).Println()
			vit.RefreshTokens()
		})
	})

	t.Run("404 on an unknown channel", func(t *testing.T) {
		url := fmt.Sprintf("api/v2/apps/test1/app1/notifications/unknownChannelID/workspaces/%d/subscriptions/app1pkg.CategoryIdx", ws.WSID)
		vit.POST(url, "",
			coreutils.WithMethod(method),
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect404(),
		).Println()
	})

	t.Run("400 on non-empty body", func(t *testing.T) {
		vit.POST(url, "some body",
			coreutils.WithMethod(method),
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect400(),
		).Println()
	})

	t.Run("400 on malformed view", func(t *testing.T) {
		url := fmt.Sprintf("api/v2/apps/test1/app1/notifications/%s/workspaces/%d/subscriptions/malformedViewQName", channelID, ws.WSID)
		vit.POST(url, "",
			coreutils.WithMethod(method),
			coreutils.WithAuthorizeBy(ws.Owner.Token),
			coreutils.Expect400(),
		).Println()
	})

	resp.HTTPResp.Body.Close()

	for range offsetsChan {
	}
	waitForDone()
}

// [~server.n10n.heartbeats/it.Heartbeat30~impl]
func TestBasicUsage_Heartbeat30(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	testProjectionKey := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1, // it does NOT matter
		Projection: in10n.QNameHeartbeat30,       // it DOES matter
		WS:         ws.WSID,                      // it does NOT matter
	}

	endCh := make(chan struct{})
	var wg sync.WaitGroup

	// Start a goroutine to simulate the passage of time
	{
		wg.Add(1)
		go func() {

			defer func() {
				wg.Done()
				close(endCh)
			}()

			for {
				select {
				case <-endCh:
					logger.Info("TestBasicUsage_Heartbeat30: endCh")
					return
				case <-time.After(100 * time.Millisecond):
					logger.Info("TestBasicUsage_Heartbeat30: MockTime.Add()")
					testingu.MockTime.Add(in10n.Heartbeat30Duration)
				}
			}

		}()
	}

	logger.Info("TestBasicUsage_Heartbeat30: before SubscribeForN10nUnsubscribe, key:", testProjectionKey)
	offsetsChan, unsubscribe := vit.SubscribeForN10nUnsubscribe(testProjectionKey)

	select {
	case <-offsetsChan:
		logger.Info("TestBasicUsage_Heartbeat30: received heartbeat notification")
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for heartbeat notification")
	}

	unsubscribe()
	endCh <- struct{}{}
	wg.Wait()
}

func TestSynthetic(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	testProjectionKey := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("paa", "price"),
		WS:         ws.WSID,
	}

	offsetsChan, unsubscribe, err := vit.N10NSubscribe(testProjectionKey)
	require.NoError(err)

	done := make(chan interface{})
	go func() {
		defer close(done)
		for offset := range offsetsChan {
			require.Equal(istructs.Offset(13), offset)
		}
	}()

	// call a test method that updates the projection
	vit.N10NUpdate(testProjectionKey, 13)

	// unsubscribe to force offsetsChan to close
	unsubscribe()

	<-done // wait for event read and offsestChan close
}

func TestChannelExpiration_V1(t *testing.T) {
	require := require.New(t)
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	testProjectionKey := in10n.ProjectionKey{
		App:        istructs.AppQName_test1_app1,
		Projection: appdef.NewQName("paa", "price"),
		WS:         ws.WSID,
	}

	offsetsChan, unsubscribe, err := vit.N10NSubscribe(testProjectionKey)
	require.NoError(err)

	// expire the channel
	testingu.MockTime.Add(25 * time.Hour)

	// produce SSE event
	vit.N10NUpdate(testProjectionKey, 13)

	// the channel is closed on SSE event because it is expired
	for range offsetsChan {
	}

	// calling unsubscribe has no sense here, it just causes "channel does not exist" error
	// but let's call for demonstration
	unsubscribe()
}

// [~server.n10n/it.AddSubscription~impl]
func TestN10NSubscribeToExtraView(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	// owning does not matter for notifications, need just a valid token
	token := ws.Owner.Token

	// subscribe to one view
	body := fmt.Sprintf(`{"subscriptions": [{"entity":"app1pkg.CategoryIdx","wsid": %d}],"expiresIn": 42}`, ws.WSID)
	resp := vit.POST("api/v2/apps/test1/app1/notifications", body,
		coreutils.WithAuthorizeBy(token),
		coreutils.WithLongPolling(),
	)

	offsetsChan, channelID, waitForDone := federation.ListenSSEEvents(resp.HTTPResp.Request.Context(), resp.HTTPResp.Body)

	// subscribe to an extra view
	body = ""
	url := fmt.Sprintf("api/v2/apps/test1/app1/notifications/%s/workspaces/%d/subscriptions/app1pkg.DailyIdx", channelID, ws.WSID)
	vit.POST(url, body,
		coreutils.WithAuthorizeBy(token),
		coreutils.WithMethod(http.MethodPut),
	)

	// force projections update
	body = `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.category","name":"Awesome food"}}]}`
	resultOffsetOfCategoryCUD := vit.PostWS(ws, "c.sys.CUD", body).CurrentWLogOffset
	body = `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.Daily","Year":42}}]}`
	resultOffsetOfDailyCUD := vit.PostWS(ws, "c.sys.CUD", body).CurrentWLogOffset

	// read events, ensure we're receiving events for the view we're additionally subscribed to
	waitForOffset(t, resultOffsetOfCategoryCUD, offsetsChan)
	waitForOffset(t, resultOffsetOfDailyCUD, offsetsChan)

	// close the initial connection
	// SSE listener channel should be closed after that
	resp.HTTPResp.Body.Close()

	for range offsetsChan {
	}
	waitForDone()
}
