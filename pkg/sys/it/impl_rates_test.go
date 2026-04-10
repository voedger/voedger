/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sys_it

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/router"
	it "github.com/voedger/voedger/pkg/vit"
)

func TestRates_BasicUsage(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	bodyQry := `{"args":{}, "elements":[{"path":"","fields":["Fld"]}]}`
	bodyCmd := `{"args":{}}`

	// first 2 calls are ok
	for i := 0; i < 2; i++ {
		vit.PostWS(ws, "q.app1pkg.RatedQry", bodyQry)
		vit.PostWS(ws, "c.app1pkg.RatedCmd", bodyCmd)
	}

	// 3rd is failed because per-minute rate is exceeded
	vit.PostWS(ws, "q.app1pkg.RatedQry", bodyQry, httpu.Expect429())
	vit.PostWS(ws, "c.app1pkg.RatedCmd", bodyCmd, httpu.Expect429())

	// proceed to the next minute to restore per-minute rates
	vit.TimeAdd(time.Minute)

	// next 2 calls are ok again
	for i := 0; i < 2; i++ {
		vit.PostWS(ws, "q.app1pkg.RatedQry", bodyQry)
		vit.PostWS(ws, "c.app1pkg.RatedCmd", bodyCmd)
	}

	// next are failed again because per-minute rate is exceeded again
	vit.PostWS(ws, "q.app1pkg.RatedQry", bodyQry, httpu.Expect429())
	vit.PostWS(ws, "c.app1pkg.RatedCmd", bodyCmd, httpu.Expect429())

	// proceed to the next minute to restore per-minute rates
	vit.TimeAdd(time.Minute)

	// next are failed again because per-hour rate is exceeded
	vit.PostWS(ws, "q.app1pkg.RatedQry", bodyQry, httpu.Expect429())
	vit.PostWS(ws, "c.app1pkg.RatedCmd", bodyCmd, httpu.Expect429())

	// proceed to the next hour to restore per-hour rates
	vit.TimeAdd(time.Hour)

	// next 2 calls are ok again
	for i := 0; i < 2; i++ {
		vit.PostWS(ws, "q.app1pkg.RatedQry", bodyQry)
		vit.PostWS(ws, "c.app1pkg.RatedCmd", bodyCmd)
	}
}

func TestRates_PerIP(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()

	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	bodyQry := `{"args":{}, "elements":[{"path":"","fields":["Fld"]}]}`
	bodyCmd := `{"args":{}}`

	for range 2 {
		vit.PostWS(ws, "q.app1pkg.IPRatedQry", bodyQry)
		vit.PostWS(ws, "c.app1pkg.IPRatedCmd", bodyCmd)
	}

	vit.PostWS(ws, "q.app1pkg.IPRatedQry", bodyQry, httpu.Expect429())
	vit.PostWS(ws, "c.app1pkg.IPRatedCmd", bodyCmd, httpu.Expect429())

	vit.TimeAdd(time.Minute)

	for range 2 {
		vit.PostWS(ws, "q.app1pkg.IPRatedQry", bodyQry)
		vit.PostWS(ws, "c.app1pkg.IPRatedCmd", bodyCmd)
	}

	vit.PostWS(ws, "q.app1pkg.IPRatedQry", bodyQry, httpu.Expect429())
	vit.PostWS(ws, "c.app1pkg.IPRatedCmd", bodyCmd, httpu.Expect429())
}

func TestQueryLimiter_BasicUsage(t *testing.T) {
	vit := it.NewVIT(t, &it.SharedConfig_App1)
	defer vit.TearDown()
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	sys := vit.GetSystemPrincipal(istructs.AppQName_test1_app1)
	limit := vit.VVMConfig.RouterMaxQueriesPerWS

	t.Run("queries rejected with 503 when per-workspace limit reached", func(t *testing.T) {
		t.Run("qpv1", func(t *testing.T) {
			wg, okToFinish := fillQuerySlots(t, vit, ws, limit)
			defer releaseQuerySlots(wg, okToFinish, limit)

			body := `{"args": {"Input": "world"},"elements": [{"fields": ["Res"]}]}`
			vit.PostWS(ws, "q.app1pkg.MockQry", body, httpu.Expect503(), httpu.WithAuthorizeBy(sys.Token), httpu.WithNoRetryPolicy())
		})

		t.Run("qpv2", func(t *testing.T) {
			wg, okToFinish := fillQuerySlots(t, vit, ws, limit)
			defer releaseQuerySlots(wg, okToFinish, limit)

			resp := vit.GET(fmt.Sprintf(`api/v2/apps/test1/app1/workspaces/%d/queries/sys.Echo?args=%s`, ws.WSID, url.QueryEscape(`{"Text":"Hello"}`)),
				httpu.WithAuthorizeBy(sys.Token), httpu.Expect503(), httpu.WithNoRetryPolicy())
			require.Equal(t, http.StatusServiceUnavailable, resp.HTTPResp.StatusCode)
			require.Equal(t, fmt.Sprintf("%d", router.DefaultRetryAfterSecondsOn503), resp.HTTPResp.Header.Get("Retry-After"))
		})
	})

	t.Run("commands not affected by query limiter", func(t *testing.T) {
		wg, okToFinish := fillQuerySlots(t, vit, ws, limit)
		defer releaseQuerySlots(wg, okToFinish, limit)

		cmdBody := `{"cuds":[{"fields":{"sys.ID":1,"sys.QName":"app1pkg.air_table_plan","name":"test"}}]}`
		vit.PostWS(ws, "c.sys.CUD", cmdBody)
	})

	t.Run("different workspaces independently limited", func(t *testing.T) {
		wg, okToFinish := fillQuerySlots(t, vit, ws, limit-1)
		defer releaseQuerySlots(wg, okToFinish, limit-1)

		ws3 := vit.WS(istructs.AppQName_test1_app1, "test_ws3")
		body := `{"args": {"Text": "world"},"elements":[{"fields":["Res"]}]}`
		resp := vit.PostWS(ws3, "q.sys.Echo", body)
		require.Equal(t, http.StatusOK, resp.HTTPResp.StatusCode)
	})
}

func fillQuerySlots(t *testing.T, vit *it.VIT, ws *it.AppWorkspace, count int) (wg *sync.WaitGroup, okToFinish chan struct{}) {
	t.Helper()
	funcStarted := make(chan struct{})
	okToFinish = make(chan struct{})
	it.MockQryExec = func(_ string, _ istructs.ExecQueryArgs, _ istructs.ExecQueryCallback) error {
		funcStarted <- struct{}{}
		<-okToFinish
		return nil
	}
	sys := vit.GetSystemPrincipal(istructs.AppQName_test1_app1)
	body := `{"args": {"Input": "world"},"elements": [{"fields": ["Res"]}]}`
	wg = &sync.WaitGroup{}
	for range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vit.PostWS(ws, "q.app1pkg.MockQry", body, httpu.WithAuthorizeBy(sys.Token))
		}()
		<-funcStarted
	}
	return wg, okToFinish
}

func releaseQuerySlots(wg *sync.WaitGroup, okToFinish chan struct{}, count int) {
	for range count {
		okToFinish <- struct{}{}
	}
	wg.Wait()
}
