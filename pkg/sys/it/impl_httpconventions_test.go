/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sys_it

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/httpu"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/vit"
)

func TestBasicUsage_HTTPConventions(t *testing.T) {
	require := require.New(t)
	vit.MockQryExec = func(input string, _ istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) error {
		rr := &rr{res: input}
		require.NoError(callback(rr))
		return errors.New("test error")
	}
	vit.MockCmdExec = func(input string, args istructs.ExecCommandArgs) error {
		return errors.New("test error")
	}
	vit := vit.NewVIT(t, &vit.SharedConfig_App1)
	defer vit.TearDown()
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")

	t.Run("query", func(t *testing.T) {
		body := `{"args": {"Input": "world"},"elements": [{"fields": ["Res"]}]}`
		resp := vit.PostWS(ws, "q.app1pkg.MockQry", body, httpu.Expect500())
		require.Equal("world", resp.SectionRow()[0])
		require.Equal(httpu.ContentType_ApplicationJSON, resp.HTTPResp.Header["Content-Type"][0])
		require.Equal(http.StatusOK, resp.HTTPResp.StatusCode)
		resp.Println()
	})

	t.Run("command", func(t *testing.T) {
		body := `{"args": {"Input": "1"}}`
		resp := vit.PostWS(ws, "c.app1pkg.MockCmd", body, httpu.Expect500())
		require.Equal(httpu.ContentType_ApplicationJSON, resp.HTTPResp.Header["Content-Type"][0])
		require.Equal(http.StatusInternalServerError, resp.HTTPResp.StatusCode)
		var sysErr coreutils.SysError
		require.ErrorAs(resp.SysError, &sysErr)
		require.Equal(http.StatusInternalServerError, sysErr.HTTPStatus)
		require.Empty(sysErr.Data)
		require.Empty(sysErr.QName)
		resp.Println()
	})
}

func TestFunctionRequestBodySizeLimit(t *testing.T) {
	const (
		requestBodySizeLimit = 200_000
		validBody            = `{"args":{"Text":"ok"},"elements":[{"fields":["Res"]}]}`
		overflowResponse     = `{"status":413,"message":"request body size limit exceeded"}`
	)
	require := require.New(t)
	vit := vit.NewVIT(t, &vit.SharedConfig_App1)
	defer vit.TearDown()
	ws := vit.WS(istructs.AppQName_test1_app1, "test_ws")
	url := fmt.Sprintf("api/test1/app1/%d/q.sys.Echo", ws.WSID)

	makeBody := func(size int) string {
		body := validBody + strings.Repeat(" ", size-len(validBody))
		require.Len([]byte(body), size)
		return body
	}

	t.Run("accepts request at limit", func(t *testing.T) {
		resp := vit.POST(url, makeBody(requestBodySizeLimit))
		require.Equal(http.StatusOK, resp.HTTPResp.StatusCode)
		require.JSONEq(`{"sections":[{"type":"","elements":[[[["ok"]]]]}]}`, resp.Body)
	})

	t.Run("rejects request above limit", func(t *testing.T) {
		resp := vit.POST(url, makeBody(requestBodySizeLimit+1), httpu.WithExpectedCode(http.StatusRequestEntityTooLarge))
		require.Equal(http.StatusRequestEntityTooLarge, resp.HTTPResp.StatusCode)
		require.JSONEq(overflowResponse, resp.Body)
	})
}
